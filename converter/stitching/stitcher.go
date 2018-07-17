package stitching

import (
	"net"
	"sync"

	"github.com/pkg/errors"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	mgo "gopkg.in/mgo.v2"
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
	sessionsColl         *mgo.Collection
	sessionsOut          chan<- *session.Aggregate
	errs                 chan<- error
	input                chan ipfix.Flow
	//inputDrained is used to keep track of this stitcher's progress
	//through its input buffer
	inputDrained *sync.WaitGroup
}

//newStitcher creates a new stitcher which uses the sessionsColl
//to match flows into session aggregates
func newStitcher(id, bufferSize int, sameSessionThreshold int64,
	sessionsColl *mgo.Collection, sessionsOut chan<- *session.Aggregate,
	errs chan<- error) *stitcher {
	return &stitcher{
		id:                   id,
		sameSessionThreshold: sameSessionThreshold,
		sessionsColl:         sessionsColl,
		sessionsOut:          sessionsOut,
		errs:                 errs,
		input:                make(chan ipfix.Flow, bufferSize),
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

	s.sessionsColl.Database.Session.Close()
	//let the manager know this stitcher is finished processing flows.
	stitcherDone.Done()
}

//enqueue inserts a flow into the input collection to be processed
//by the loop in start
func (s *stitcher) enqueue(flow ipfix.Flow) {
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
//uses the sessionsColl as a lookup table to match flows
//against each other. Once a flow has been matched in both
//directions, the resulting session aggregate is sent to
//the sessionsOut channel.
func (s *stitcher) stitchFlow(flow ipfix.Flow) error {
	//Create a session aggregate from the flow
	var sessAgg session.Aggregate
	err := session.FromFlow(flow, &sessAgg)
	if err != nil {
		return errors.Wrap(err, "could not create session.Aggregate from flow")
	}

	//We don't know how to stitch everything under the sun
	//Unkown protocols and special addresses may cause us to bail on stitching
	if s.shouldSkipStitching(flow) {
		s.sessionsOut <- &sessAgg
		return nil
	}

	//matchFound is true when another session is found with the same
	//AggregateQuery in the sessions collection, and the
	//sessions qualify for merging/ stitching
	matchFound := false

	var oldSessAgg session.Aggregate
	oldSessAggIter := s.sessionsColl.Find(&sessAgg.AggregateQuery).Iter()

	//TODO: stitch with flow with closest timestamps rather than
	//taking the first one that matches

	//iterate over the possible matches
	for oldSessAggIter.Next(&oldSessAgg) && !matchFound {
		//its possible these flows shouldn't be merged based on timestamps
		//and FlowEndReasons
		if s.shouldMerge(&sessAgg, &oldSessAgg) {
			matchFound = true

			//do the actual merge
			err = sessAgg.Merge(&oldSessAgg)
			if err != nil {
				return errors.Wrapf(err, "cannot merge session\n%+v\nwith\n%+v", &sessAgg, &oldSessAgg)
			}

			//if both sides of the session have been filled, write it out
			if sessAgg.FilledFromSourceA && sessAgg.FilledFromSourceB {
				err := s.sessionsColl.RemoveId(oldSessAgg.ID)
				if err != nil {
					return errors.Wrap(err, "could not remove old session aggregate")
				}
				s.sessionsOut <- &sessAgg
			} else {
				//otherwise update the database
				err := s.sessionsColl.UpdateId(oldSessAgg.ID, &sessAgg)
				if err != nil {
					return errors.Wrap(err, "could not update existing session aggregate")
				}
			}
		}
	}

	//if there's an error other than not found, return it up
	if oldSessAggIter.Err() != nil && oldSessAggIter.Err() != mgo.ErrNotFound {
		return errors.Wrap(oldSessAggIter.Err(), "could not find all matching session aggregates")
	}

	//no matching unstitched session found
	if !matchFound {
		err := s.sessionsColl.Insert(&sessAgg)
		if err != nil {
			return errors.Wrap(err, "could not insert new session aggregate")
		}
	}
	return nil
}

//shouldMerge details whether or not two session.Aggregate objects
//should be merged with each other. These requirements go beyond
//having a matching AggregateQuery. They are largely based
//on timestamps and flow end reasons
func (s *stitcher) shouldMerge(newSessAgg *session.Aggregate, oldSessAgg *session.Aggregate) bool {

	if oldSessAgg.ProtocolIdentifier == protocols.TCP && (newSessAgg.FilledFromSourceA && oldSessAgg.FlowEndReasonAB == ipfix.EndOfFlow ||
		newSessAgg.FilledFromSourceB && oldSessAgg.FlowEndReasonBA == ipfix.EndOfFlow) {
		return false
	}

	//grab the latest FlowEnd from the new session aggregate
	newSessAggFlowEnd := newSessAgg.FlowEndMillisecondsAB
	if newSessAgg.FlowEndMillisecondsBA > newSessAggFlowEnd {
		newSessAggFlowEnd = newSessAgg.FlowEndMillisecondsBA
	}

	//grab the earliest FlowStart from the new session aggregate
	newSessAggFlowStart := newSessAgg.FlowStartMillisecondsAB
	if newSessAggFlowStart == 0 || newSessAgg.FlowStartMillisecondsBA != 0 &&
		newSessAgg.FlowStartMillisecondsBA < newSessAggFlowStart {
		newSessAggFlowStart = newSessAgg.FlowStartMillisecondsBA
	}

	oldSessAggMinFlowEnd := newSessAggFlowStart - s.sameSessionThreshold
	oldSessAggMaxFlowStart := newSessAggFlowEnd + s.sameSessionThreshold

	//grab the latest FlowEnd from the old session aggregate
	oldSessAggFlowEnd := oldSessAgg.FlowEndMillisecondsAB
	if oldSessAgg.FlowEndMillisecondsBA > oldSessAggFlowEnd {
		oldSessAggFlowEnd = oldSessAgg.FlowEndMillisecondsBA
	}

	//grab the earliest FlowStart from the old session aggregate
	oldSessAggFlowStart := oldSessAgg.FlowStartMillisecondsAB
	if oldSessAggFlowStart == 0 || oldSessAgg.FlowStartMillisecondsBA != 0 &&
		oldSessAgg.FlowStartMillisecondsBA < oldSessAggFlowStart {
		oldSessAggFlowStart = oldSessAgg.FlowStartMillisecondsBA
	}

	return oldSessAggFlowStart <= oldSessAggMaxFlowStart &&
		oldSessAggFlowEnd >= oldSessAggMinFlowEnd
}

//shouldSkipStitching determines whether or not we know
//how to stitch a given flow. Protocols and special addresses
//determine whether or not stitching is possible.
func (s *stitcher) shouldSkipStitching(flow ipfix.Flow) bool {
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
func (s *stitcher) destIsMulticastOrBroadcast(flow ipfix.Flow) bool {
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
