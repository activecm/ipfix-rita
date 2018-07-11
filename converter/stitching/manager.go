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
}

//NewManager creates a Manager with the given settings
func NewManager(sameSessionThreshold int64, numStitchers int32,
	stitcherBufferSize, outputBufferSize int) Manager {

	return Manager{
		sameSessionThreshold: sameSessionThreshold,
		stitcherBufferSize:   stitcherBufferSize,
		numStitchers:         numStitchers,
		outputBufferSize:     outputBufferSize,
	}
}

//RunSync converts an ordered array of ipfix.Flow objects
//into an unordered array of *session.Aggregates.
//An active connection to MongoDB is needed for this process.
//This function is a synchronous wrapper around RunAsync.
func (m Manager) RunSync(input []ipfix.Flow, db database.DB) ([]*session.Aggregate, []error) {
	//feed the input input a channel for the manager
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
	sessionsMutex := new(sync.Mutex)
	var errs []error
	errsMutex := new(sync.Mutex)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		for sessionAggregate := range sessionsChan {
			sessionsMutex.Lock()
			sessions = append(sessions, sessionAggregate)
			sessionsMutex.Unlock()
		}
		wg.Done()
	}()

	go func() {
		for err := range errsChan {
			errsMutex.Lock()
			errs = append(errs, err)
			errsMutex.Unlock()
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

	//In order to parallelize the stitching process, we need to maintain
	//relative order for each thread. Hash partitioning ensures no two
	//stitchers will work on the same session.AggregateQuery. Additionally,
	//since the data comes in in order from input, each stitcher sees
	//ordered data.
	partitions := make([]chan ipfix.Flow, 0, m.numStitchers) // input channels

	//We use the stichersDone WaitGroup to wait for the stitchers to finish
	stitchersDone := new(sync.WaitGroup)

	for i := 0; i < int(m.numStitchers); i++ {
		partitions = append(partitions, make(chan ipfix.Flow, m.stitcherBufferSize))

		//create and start the stitchers
		stitchersDone.Add(1)
		//newStitcher(sticherID, sameSessionThreshold)
		go newStitcher(i, m.sameSessionThreshold).run(
			partitions[i], db.NewSessionsConnection(),
			sessions, errs, stitchersDone,
		)
	}

	var flowCount int

	//loop over the input until its closed
	//If the input is coming from ipfix.mgologstash, the input channel will
	//be closed when the program recieves CTRL-C
	for inFlow := range input {
		flowCount++
		stitcherID := m.selectStitcher(inFlow)
		//Send the flow to the assigned stitcher
		//This may block if the stitcher's buffer is full.
		partitions[stitcherID] <- inFlow
	}

	//Let the stitchers know no more data is coming
	for i := range partitions {
		close(partitions[i])
	}
	//Wait for the stitchers to exit
	stitchersDone.Wait()

	//flush the rest of the sessions out
	var unstitchedCount int
	sessionsColl := db.NewSessionsConnection()
	unstitchedSessionsIter := sessionsColl.Find(nil).Iter()
	var sessionAgg session.Aggregate
	for unstitchedSessionsIter.Next(&sessionAgg) {
		unstitchedCount++
		sessions <- &sessionAgg
		err := sessionsColl.RemoveId(sessionAgg.ID)
		if err != nil {
			errs <- err
		}
	}
	if unstitchedSessionsIter.Err() != nil {
		errs <- unstitchedSessionsIter.Err()
	}

	sessionsColl.Database.Session.Close()

	fmt.Printf("Flows Read: %d\n", flowCount)
	fmt.Printf("Flows Left Unstitched: %d\n", unstitchedCount)
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

	partition := int32(hasher.Sum32()) % m.numStitchers
	if partition < 0 {
		partition = -partition
	}
	return int(partition)
}
