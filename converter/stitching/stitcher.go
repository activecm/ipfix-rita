package stitching

import (
	"net"
	"sync"

	"github.com/pkg/errors"

	"math"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/matching"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
)

//v4MulticastNet represents all IPv4 multicast addresses
var _, v4MulticastNet, _ = net.ParseCIDR("224.0.0.0/4")

//v4BroadcastIP represents the all-hosts IPv4 broadcast address
var v4BroadcastIP = net.ParseIP("255.255.255.255")

//v6MulticastNet represents all IPv6 multicast addresses
//in IPv6 multicast has completely replaced broadcast
var _, v6MulticastNet, _ = net.ParseCIDR("FF00::/8")

//stitcher is the main worker for stitching.Manager
type stitcher struct {
	id                   int
	sameSessionThreshold int64
	matcher              matching.Matcher
	sessionsOut          chan<- *session.Aggregate
	errs                 chan<- error
	input                chan input.Flow
	//inputDrained is used to keep track of this stitcher's progress
	//through its input buffer
	inputDrained *sync.WaitGroup
}

//newStitcher creates a new stitcher which uses the matcher
//to match flows into session aggregates
func newStitcher(id int, bufferSize int64, sameSessionThreshold int64,
	matcher matching.Matcher, sessionsOut chan<- *session.Aggregate,
	errs chan<- error) *stitcher {
	return &stitcher{
		id:                   id,
		sameSessionThreshold: sameSessionThreshold,
		matcher:              matcher,
		sessionsOut:          sessionsOut,
		errs:                 errs,
		input:                make(chan input.Flow, bufferSize),
		inputDrained:         new(sync.WaitGroup),
	}
}

//run processes flows in the input channel as provided by the
//enqueue method.
func (s *stitcher) run(stitcherDone *sync.WaitGroup) {
	for inFlow := range s.input {
		err := s.stitchFlow(inFlow)
		if err != nil {
			s.errs <- errors.Wrapf(err, "error stitching %+v", inFlow)
		}
		s.inputDrained.Done()
	}

	//let the manager know this stitcher is finished processing flows.
	stitcherDone.Done()
}

//enqueue inserts a flow into the input collection to be processed
//by the loop in start
func (s *stitcher) enqueue(flow input.Flow) {
	s.inputDrained.Add(1)
	s.input <- flow
}

//beginShutdown closes the internal input channel which
//breaks the main loop in start() allowing the method
//to exit
func (s *stitcher) beginShutdown() {
	close(s.input)
}

//waitForFlush waits for this stitcher to process the flows
//that have been enqueued in its input buffer
func (s *stitcher) waitForFlush() {
	s.inputDrained.Wait()
}

//stitchFlow implements the main stitching logic. The method
//uses the matcher as a lookup table to match flows
//against each other. Once a flow has been matched in both
//directions, the resulting session aggregate is sent to
//the sessionsOut channel.
func (s *stitcher) stitchFlow(flow input.Flow) error {
	//Create a session aggregate from the flow
	var newSessAgg session.Aggregate
	err := session.FromFlow(flow, &newSessAgg)
	if err != nil {
		return errors.Wrap(err, "could not create session.Aggregate from flow")
	}

	//We don't know how to stitch everything under the sun
	//Unkown protocols and special addresses may cause us to bail on stitching
	if s.shouldSkipStitching(flow) {
		s.sessionsOut <- &newSessAgg
		return nil
	}

	//matchFound is true when another session is found with the same
	//AggregateQuery in the matcher, and the
	//sessions qualify for merging/ stitching
	var matchFound = false
	var matchAgg session.Aggregate
	matchCost := int64(math.MaxInt64)

	var oldSessAgg session.Aggregate
	oldSessAggIter := s.matcher.Find(&newSessAgg.AggregateQuery)
	//iterate over the possible matches
	for oldSessAggIter.Next(&oldSessAgg) {
		//its possible these flows shouldn't be merged based on timestamps
		//and FlowEndReasons
		if s.shouldMerge(&newSessAgg, &oldSessAgg) {

			var diff1 = newSessAgg.FlowEndMilliseconds() - oldSessAgg.FlowEndMilliseconds()
			if diff1 < 0 {
				diff1 *= -1
			}
			var diff2 = newSessAgg.FlowStartMilliseconds() - oldSessAgg.FlowStartMilliseconds()
			if diff2 < 0 {
				diff2 *= -1
			}

			newMatchCost := diff1 + diff2
			if newMatchCost < matchCost {
				matchFound = true
				matchCost = newMatchCost
				matchAgg = oldSessAgg
			}
		}
	}

	if matchFound {
		err = newSessAgg.Merge(&matchAgg)
		if err != nil {
			return errors.Wrapf(err, "cannot merge session\n%+v\nwith\n%+v", &newSessAgg, &matchAgg)
		}
		if newSessAgg.FilledFromSourceA && newSessAgg.FilledFromSourceB { //The session has both sides of the connection detailed
			err := s.matcher.Remove(&matchAgg)
			if err != nil {
				return errors.Wrap(err, "could not remove old session aggregate")
			}
			s.sessionsOut <- &newSessAgg
		} else {
			//The merge happened on the same side of the connection
			//The newly merged connection needs to replace the old connection in the matcher
			//Merge doesn't carry the MatcherID through. We need to set the MatcherID
			//so the Update method updates the right session aggregate.
			newSessAgg.MatcherID = matchAgg.MatcherID
			err := s.matcher.Update(&newSessAgg)
			if err != nil {
				return errors.Wrap(err, "could not update existing session aggregate")
			}
		}
	} else {
		err := s.matcher.Insert(&newSessAgg)
		if err != nil {
			return errors.Wrap(err, "could not insert session aggregate")
		}
	}

	return nil
}

//shouldMerge details whether or not two session.Aggregate objects
//should be merged with each other. These requirements go beyond
//having a matching AggregateQuery. They are largely based
//on timestamps and flow end reasons
func (s *stitcher) shouldMerge(newSessAgg *session.Aggregate, oldSessAgg *session.Aggregate) bool {

	if oldSessAgg.ProtocolIdentifier == protocols.TCP && (newSessAgg.FilledFromSourceA && oldSessAgg.FlowEndReasonAB == input.EndOfFlow ||
		newSessAgg.FilledFromSourceB && oldSessAgg.FlowEndReasonBA == input.EndOfFlow) {
		return false
	}

	return oldSessAgg.FlowStartMilliseconds() <=
		(newSessAgg.FlowEndMilliseconds()+s.sameSessionThreshold) &&
		oldSessAgg.FlowEndMilliseconds() >=
			(newSessAgg.FlowStartMilliseconds()-s.sameSessionThreshold)

}

//shouldSkipStitching determines whether or not we know
//how to stitch a given flow. Protocols and special addresses
//determine whether or not stitching is possible.
func (s *stitcher) shouldSkipStitching(flow input.Flow) bool {
	//If the destination is multicast or broadcast,
	//write the flow out without stitching
	if s.destIsMulticastOrBroadcast(flow) {
		return true
	}

	//We only know how to stitch TCP and UDP
	//If the protocol is something out, write it out without stitching
	if flow.ProtocolIdentifier() != protocols.TCP && flow.ProtocolIdentifier() != protocols.UDP {
		return true
	}

	return false
}

//destIsMulticastOrBroadcast determines whether the destination
//of a flow is a multicast or broadcast IPv4/ IPv6 address
func (s *stitcher) destIsMulticastOrBroadcast(flow input.Flow) bool {
	destIP := net.ParseIP(flow.DestinationIPAddress())
	if destIP.To4() != nil {
		//unfortunately we can't check for network specific broadcast addresses
		//since we don't know the network layout
		if v4MulticastNet.Contains(destIP) || destIP.Equal(v4BroadcastIP) {
			return true
		}
	} else {
		if v6MulticastNet.Contains(destIP) {
			return true
		}
	}
	return false
}
