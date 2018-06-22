package stitching

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/output"
)

type Manager struct {
	sameSessionThreshold uint64
	stitcherBufferSize   int
	numStitchers         int32
}

func NewManager(sameSessionThreshold uint64, stitcherBufferSize int,
	numStitchers int32) Manager {

	return Manager{
		sameSessionThreshold: sameSessionThreshold,
		stitcherBufferSize:   stitcherBufferSize,
		numStitchers:         numStitchers,
	}
}

func (m Manager) RunAsync(input <-chan ipfix.Flow,
	db database.DB, writer output.SessionWriter) <-chan error {
	errs := make(chan error)
	go m.runInner(input, errs, db, writer)
	return errs
}

func (m Manager) runInner(input <-chan ipfix.Flow, errs chan<- error,
	db database.DB, writer output.SessionWriter) {

	//Create a map of IPFIX/ netflow exporters
	//Each exporter has a flusher object which handles
	//removing expired sessions from the sessions collection
	exporters := newExporterMap()

	//We need some synchronization primitives to interact with each
	//exporter's flusher
	flusherContext := context.Background()
	flusherContext, cancelFlushers := context.WithCancel(flusherContext)
	var flusherDoneSignals []chan struct{}

	//In order to parallelize the stitching process, we need to maintain
	//relative order for each thread. Hash partitioning ensures no two
	//stitchers will work on the same session.AggregateQuery. Additionally,
	//since the data comes in in order from input, each stitcher sees
	//ordered data.
	partitions := make([]chan ipfix.Flow, 0, m.numStitchers)        // input channel
	stitcherDoneSignals := make([]chan struct{}, 0, m.numStitchers) // synchro
	stitchers := make([]stitcher, 0, m.numStitchers)
	for i := 0; i < int(m.numStitchers); i++ {
		partitions = append(partitions, make(chan ipfix.Flow, m.stitcherBufferSize))
		stitcherDoneSignals = append(stitcherDoneSignals, make(chan struct{}))
		stitchers = append(stitchers, newStitcher(i, m.sameSessionThreshold))

		//start the stitchers
		go stitchers[i].run(
			partitions[i], errs, stitcherDoneSignals[i],
			exporters,
			db.NewSessionsConnection(),
			writer,
		)
	}

	//loop over the input until its closed
	//If the input is coming from ipfix.mgologstash, the input channel will
	//be closed when the program recieves CTRL-C
	for inFlow := range input {
		stitcherID := m.selectStitcher(inFlow)

		//Flows which ended before the "maxExpireTime" are not needed for this
		//flow and can, therefore, be expired out. However, other flows
		//currently queued up may need data before this maxExpireTime.
		//The minimum maxExpireTime across stitchers is used as the expiration clock
		//per exporter.

		flowStartTime, err := inFlow.FlowStartMilliseconds()
		if err != nil { //parsing errors (shouldn't)* happen
			errs <- err
			continue
		}

		//The maxExpireTime is calulated by subtracting the sameSessionThreshold
		//from this flow's flowStartTime. The maxExpireTime is compared against
		//the FlowEnd times of the previous flows.
		//
		//As a thought experiment, replace maxExpireTime with "oldFlow.FlowEnd"
		//and replace the equals sign with a less than sign.
		//Then, add sameSessionThreshold to both sides.
		//Now, we have oldFlow.FlowEnd + sameSessionThreshold < newFlow.FlowStart.
		//Finally, invert the condition so we are checking for non-expired
		//flows. We end up with oldFlow.FlowEnd + sameSessionThreshold >= newFlow.FlowStart.
		//(newFlow.flowStart <= oldFlow.FlowEnd + sameSessionThreshold)
		//Which is the key condition in deciding whether or not to merge two flows.

		maxExpireTime := flowStartTime - m.sameSessionThreshold

		//cache the exporter address since we use it a number of times coming up
		exporterAddress := inFlow.Exporter()

		//We record the maxExpireTime here since total order is lost
		//once the data goes to the stitcher. Order is only maintained per
		//partition. For example, due to the nature of hash partitioning,
		//one stitcher may take on more work than its peers for a short amount of time.
		//The flows in its buffer have lesser maxExpireTimes than the subsequent
		//flows assigned to its peers.
		exporter, ok := exporters.get(exporterAddress)

		//This is the first time we've seen this exporter
		if !ok {
			//Create a new exporter and register it in the map
			exporter = newExporter(exporterAddress)
			exporters.add(exporter)

			//Launch a flusher to handle removing expired session aggregates
			flusherDoneSignals = append(flusherDoneSignals, make(chan struct{}))

			go exporter.flusher.run(
				flusherContext,
				flusherDoneSignals[len(flusherDoneSignals)-1],
				db.NewSessionsConnection(),
				writer,
			)
		}

		exporter.flusher.appendMaxExpireTime(stitcherID, maxExpireTime)

		//Send the flow to the assigned stitcher
		//This may block if the stitcher's buffer is full.
		partitions[stitcherID] <- inFlow
	}

	//Let the stitchers know no more data is coming
	for i := range partitions {
		close(partitions[i])
	}
	//Wait for the stitchers to exit
	for i := range stitcherDoneSignals {
		<-stitcherDoneSignals[i]
	}
	//Stop the flushers since no more data is being created
	//This should trigger a full flush of the database
	cancelFlushers()
	//Wait for flushers to finish
	for i := range flusherDoneSignals {
		<-flusherDoneSignals[i]
	}
	//all senders on the errors channel have finished execution
	close(errs)
	fmt.Println("Stitching manager finished")
}

func (m Manager) selectStitcher(f ipfix.Flow) int {
	hasher := fnv.New32()
	hasher.Write([]byte(f.Exporter()))
	hasher.Write([]byte(f.SourceIPAddress()))
	hasher.Write([]byte(f.DestinationIPAddress()))

	var buffer [2]byte
	bufferSlice := buffer[:]

	binary.LittleEndian.PutUint16(bufferSlice, f.SourcePort())
	hasher.Write(bufferSlice)

	binary.LittleEndian.PutUint16(bufferSlice, f.DestinationPort())
	hasher.Write(bufferSlice)

	bufferSlice = buffer[:1]
	bufferSlice[0] = uint8(f.ProtocolIdentifier())
	hasher.Write(bufferSlice)

	partition := int32(hasher.Sum32()) % m.numStitchers
	if partition < 0 {
		partition = -partition
	}
	return int(partition)
}
