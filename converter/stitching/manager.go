package stitching

import (
	"encoding/binary"
	"hash/fnv"
	"sync"

	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/pkg/errors"
)

//Manager stitches together a series of ipfix.Flow objects into
//*session.Aggregate objects.
type Manager struct {
	//sameSessionThreshold determines whether two flows are part of the same session
	//when there is no clear way to decide. For example, if a UDP connection
	//starts after a previous connection ended with the same Flow Key, within the
	//sameSessionThreshold, the two connections will be treated as a single session.
	sameSessionThreshold int64
	//numStitchers determines how many workers should process flows at at time
	numStitchers int32
	//stitcherBufferSize determines how many input flows should be buffered for
	//each stitcher
	stitcherBufferSize int
	//outputBufferSize determines how many output session aggregates should
	//be buffered overall
	outputBufferSize int
	//sessionsTableMaxSize determines the max amount of unmatched session aggregates
	//that may exist in the sessions table/collection before a flush happens
	sessionsTableMaxSize int
	log                  logging.Logger
}

//NewManager creates a Manager with the given settings
func NewManager(sameSessionThreshold int64, numStitchers int32,
	stitcherBufferSize, outputBufferSize, sessionsTableMaxSize int,
	log logging.Logger) Manager {

	return Manager{
		sameSessionThreshold: sameSessionThreshold,
		stitcherBufferSize:   stitcherBufferSize,
		numStitchers:         numStitchers,
		outputBufferSize:     outputBufferSize,
		sessionsTableMaxSize: sessionsTableMaxSize,
		log:                  log,
	}
}

//RunSync converts an ordered array of ipfix.Flow objects
//into an unordered array of *session.Aggregates.
//An active connection to MongoDB is needed for this process.
//This function is a synchronous wrapper around RunAsync.
func (m Manager) RunSync(input []ipfix.Flow, db database.DB) ([]*session.Aggregate, []error) {
	//run the input array through a channel for the runAsync method
	inputChan := make(chan ipfix.Flow)
	go func() {
		for i := range input {
			inputChan <- input[i]
		}
		close(inputChan)
	}()

	//grab the results from the async method
	sessionsChan, errsChan := m.RunAsync(inputChan, db)

	//append the results to the output buffers
	var sessions []*session.Aggregate
	var errs []error

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		for sessionAggregate := range sessionsChan {
			sessions = append(sessions, sessionAggregate)
		}
		wg.Done()
	}()

	go func() {
		for err := range errsChan {
			errs = append(errs, err)
		}
		wg.Done()
	}()

	wg.Wait()
	return sessions, errs
}

//RunAsync converts an ordered stream of ipfix.Flow objects
//into an unordered stream of *session.Aggregates.
//An active connection to MongoDB is needed for this process.
func (m Manager) RunAsync(input <-chan ipfix.Flow,
	db database.DB) (<-chan *session.Aggregate, <-chan error) {
	errs := make(chan error)
	sessions := make(chan *session.Aggregate, m.outputBufferSize)
	go m.runInner(input, db, sessions, errs)
	return sessions, errs
}

//runInner implements the bulk of RunAsync
func (m Manager) runInner(input <-chan ipfix.Flow, db database.DB,
	sessions chan<- *session.Aggregate, errs chan<- error) {

	//In order to parallelize the stitching process, we use hash partitioning
	//which ensures no two stitchers will work on the same session.AggregateQuery.

	//Initialize the stitchers and start them off
	stitchers := make([]*stitcher, m.numStitchers)

	//We use the stichersDone WaitGroup to wait for the stitchers to finish
	stitchersDone := new(sync.WaitGroup)

	for i := 0; i < int(m.numStitchers); i++ {
		//create and start the stitchers
		stitchers[i] = newStitcher(i, m.stitcherBufferSize, m.sameSessionThreshold, db.NewSessionsConnection(), sessions, errs)
		stitchersDone.Add(1)
		go stitchers[i].run(stitchersDone)
	}

	//flusher ensures the sessions collection/table never
	//exceeds a constant size and is responsible for flushing out
	//flows which were never matched with other flows
	flusher := newFlusher(
		db.NewSessionsConnection(),
		sessions,
		m.sessionsTableMaxSize,
		0.9, //.9 means flush down to .9 * m.sessionsTableMaxSize aggregates
	)

	//keep track of how many flows we process
	var flowCount int

	//loop over the input until its closed
	//If the input is coming from ipfix.mgologstash and managed by
	//convert.go, the input channel will
	//be closed when the program recieves CTRL-C
	for inFlow := range input {
		flowCount++
		//use the hash partitioner to assign the flow to a stitcher
		stitcherID := m.selectStitcher(inFlow)
		//Send the flow to the assigned stitcher
		//This may block if the stitcher's buffer is full.
		stitchers[stitcherID].enqueue(inFlow)

		//check if the sessions collection is too full
		//TODO: Find a better way to monitor the size of the table
		//instead of polling on every flow.
		//maybe just mod flowCount and check every so often
		shouldFlush, err := flusher.shouldFlush()
		if err != nil {
			errs <- errors.Wrap(err, "could not check whether the sessions collection should be flushed")
			continue //we can't trust shouldFlush if there is an error
		}

		if shouldFlush {
			m.log.Info("initiating session aggregate flush", nil)
			//wait for the stitchers to run through their buffers
			for i := 0; i < int(m.numStitchers); i++ {
				stitchers[i].waitForFlush()
			}

			err := flusher.flush()
			if err != nil {
				errs <- errors.Wrap(err, "could not flush the sessions collection")
			}
		}
	}

	//Start shutting down the the stitchers
	for i := range stitchers {
		stitchers[i].beginShutdown()
	}
	//Wait for the stitchers to exit
	stitchersDone.Wait()

	//flush the rest of the sessions out
	flusher.flushAll()
	flusher.close()

	m.log.Info("stitching manager exiting", logging.Fields{
		"flows read": flowCount,
		"flows stitched": flowCount -
			flusher.nPacketConnsFlushed[1] -
			flusher.nPacketConnsFlushed[2] -
			flusher.oldConnsFlushed,
		"1 packet flows left unstitched": flusher.nPacketConnsFlushed[1],
		"2 packet flows left unstitched": flusher.nPacketConnsFlushed[2],
		"other flows left unstitched":    flusher.oldConnsFlushed,
	})

	//all stichers and flushers are done, no more sessions can be produced
	close(sessions)
	//all senders on the errors channel have finished execution
	close(errs)
	m.log.Info("stitching manager exited", nil)
}

//selectStitcher hashes a flow's flow key and mods the result over the
//number of stitchers
func (m Manager) selectStitcher(f ipfix.Flow) int {
	hasher := fnv.New32()
	var buffer [2]byte

	hasher.Write([]byte(f.Exporter()))

	bufferSlice := buffer[:1]
	bufferSlice[0] = uint8(f.ProtocolIdentifier())
	hasher.Write(bufferSlice)

	bufferSlice = buffer[:]

	//flows from A->B and from B->A should hash to the same value
	//We impose an order such that the alphabetically lesser ip
	//address is hashed first
	if f.SourceIPAddress() < f.DestinationIPAddress() {
		hasher.Write([]byte(f.SourceIPAddress()))

		binary.LittleEndian.PutUint16(bufferSlice, f.SourcePort())
		hasher.Write(bufferSlice)

		hasher.Write([]byte(f.DestinationIPAddress()))

		binary.LittleEndian.PutUint16(bufferSlice, f.DestinationPort())
		hasher.Write(bufferSlice)
	} else {
		hasher.Write([]byte(f.DestinationIPAddress()))

		binary.LittleEndian.PutUint16(bufferSlice, f.DestinationPort())
		hasher.Write(bufferSlice)

		hasher.Write([]byte(f.SourceIPAddress()))

		binary.LittleEndian.PutUint16(bufferSlice, f.SourcePort())
		hasher.Write(bufferSlice)
	}

	//theres no int abs function in go >.<"
	partition := int32(hasher.Sum32()) % m.numStitchers
	if partition < 0 {
		partition = -partition
	}
	return int(partition)
}
