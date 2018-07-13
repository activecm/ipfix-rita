package stitching

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
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
}

//NewManager creates a Manager with the given settings
func NewManager(sameSessionThreshold int64, numStitchers int32,
	stitcherBufferSize, outputBufferSize, sessionsTableMaxSize int) Manager {

	return Manager{
		sameSessionThreshold: sameSessionThreshold,
		stitcherBufferSize:   stitcherBufferSize,
		numStitchers:         numStitchers,
		outputBufferSize:     outputBufferSize,
		sessionsTableMaxSize: sessionsTableMaxSize,
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
		go stitchers[i].start(stitchersDone)
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
		shouldFlush, err := flusher.shouldFlush()
		if err != nil {
			errs <- err
			continue //we can't trust shouldFlush if there is an error
		}

		if shouldFlush {
			//wait for the stitchers to run through their buffers
			for i := 0; i < int(m.numStitchers); i++ {
				stitchers[i].waitForFlush()
			}

			err := flusher.flush()
			if err != nil {
				errs <- err
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

	fmt.Printf("Flows Read: %d\n", flowCount)
	fmt.Printf("1 Packet Flows Left Unstitched: %d\n", flusher.nPacketConnsFlushed[1])
	fmt.Printf("2 Packet Flows Left Unstitched: %d\n", flusher.nPacketConnsFlushed[2])
	fmt.Printf("Other Flows Left Unstitched: %d\n", flusher.oldConnsFlushed)

	//all stichers and flushers are done, no more sessions can be produced
	close(sessions)
	//all senders on the errors channel have finished execution
	close(errs)
	fmt.Println("Stitching manager finished")
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
