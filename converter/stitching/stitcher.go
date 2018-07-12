package stitching

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	mgo "gopkg.in/mgo.v2"
)

type stitcher struct {
	id                   int
	sameSessionThreshold int64
}

func newStitcher(id int, sameSessionThreshold int64) stitcher {
	return stitcher{
		id:                   id,
		sameSessionThreshold: sameSessionThreshold,
	}
}

func (s stitcher) run(input <-chan ipfix.Flow, sessionsColl *mgo.Collection,
	sessionsOut chan<- *session.Aggregate, errs chan<- error,
	stitcherDone *sync.WaitGroup) {

	for inFlow := range input {
		err := s.stitchFlow(inFlow, sessionsColl, sessionsOut)
		if err != nil {
			errs <- err
		}
	}

	sessionsColl.Database.Session.Close()
	//let the manager know this stitcher is finished processing flows.
	stitcherDone.Done()
}

func (s stitcher) stitchFlow(flow ipfix.Flow, sessionsColl *mgo.Collection, sessionsOut chan<- *session.Aggregate) error {
	//If this is a junk connection throw it out and continue
	proto := flow.ProtocolIdentifier()
	if proto == protocols.TCP && flow.PacketTotalCount() <= 2 {
		return nil
	}

	//Create a session aggregate from the flow
	var sessAgg session.Aggregate
	err := session.FromFlow(flow, &sessAgg)
	if err != nil {
		return err
	}

	//We only know how to stitch TCP and UDP
	//If the protocol is something out, write it out without stitching
	if proto != protocols.TCP && proto != protocols.UDP {
		sessionsOut <- &sessAgg
		return nil
	}

	//try to find an unstitched session with the same AggregateQuery (Flow Key + Exporter)
	//and remove it from the table if its found
	var oldSessAgg session.Aggregate
	_, err = sessionsColl.Find(&sessAgg.AggregateQuery).Apply(mgo.Change{
		Remove: true,
	}, &oldSessAgg)

	//if we don't expect the error, return it
	if err != nil && err != mgo.ErrNotFound {
		return err
	}

	//oldSessAgg successfully populated
	if err == nil {
		//if the timestamps match up and no weird EOF business is going on
		if s.shouldMerge(&sessAgg, &oldSessAgg) {

			//do the actual merge
			err = sessAgg.Merge(&oldSessAgg)
			if err != nil {
				return err
			}

			//if both sides of the session have been filled, write it out
			if sessAgg.FilledFromSourceA && sessAgg.FilledFromSourceB {
				sessionsOut <- &sessAgg
			} else {
				//otherwise update the database
				err := sessionsColl.Insert(&sessAgg)
				if err != nil {
					return err
				}
			}
		} else {
			//write out the old one-sided aggregate and replace it
			//TODO: count how many times this happens
			sessionsOut <- &oldSessAgg
			err := sessionsColl.Insert(&sessAgg)
			if err != nil {
				return err
			}
			return errors.New("session timing mismatch")
		}
	} else {
		//no unstitched session found
		err := sessionsColl.Insert(&sessAgg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s stitcher) shouldMerge(newSessAgg *session.Aggregate, oldSessAgg *session.Aggregate) bool {

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
