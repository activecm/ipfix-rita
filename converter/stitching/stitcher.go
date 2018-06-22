package stitching

import (
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	mgo "gopkg.in/mgo.v2"
)

type stitcher struct {
	id                   int
	sameSessionThreshold uint64
}

func newStitcher(id int, sameSessionThreshold uint64) stitcher {
	return stitcher{
		id:                   id,
		sameSessionThreshold: sameSessionThreshold,
	}
}

func (s stitcher) run(input <-chan ipfix.Flow, errs chan<- error,
	doneSignal chan<- struct{}, exporters exporterMap,
	sessionsColl *mgo.Collection, writer output.SessionWriter) {

	for inFlow := range input {
		//The maxExpireTime was added to the flusher for this flow
		//in stitching.Manager. It was added in the Manager
		//rather than at the beginning of the range loop since
		//there is no guarantee that this go routine will run immediately.
		//A stitcher may have a flow with a lower maxExpireTime in its input buffer
		//than the maxExpireTimes held by its peers.
		//
		//We can ignore the ok check since we know the manager created the exporter
		exporter, _ := exporters.get(inFlow.Exporter())

		err := s.stitchFlow(inFlow, sessionsColl, writer)
		if err != nil {
			errs <- err
		}

		//StitcherDone lets the flusher know that this stitcher is done
		//processing this flow. The flusher can then update this stitcher's
		//maxExpireTime with the maxExpireTime derived from the next flow
		//that this stitcher will process from the same exporter.
		exporter.flusher.stitcherDone(s.id)
	}

	//let the manager know this stitcher is finished processing flows.
	close(doneSignal)
}

func (s stitcher) stitchFlow(flow ipfix.Flow, sessionsColl *mgo.Collection, writer output.SessionWriter) error {
	//If this is a junk connection throw it out and continue
	proto := flow.ProtocolIdentifier()
	if proto == protocols.TCP && flow.PacketTotalCount() < 2 {
		return nil
	}

	//Create a session aggregate from the flow
	var sessAgg session.Aggregate
	srcMapping, err := session.FromFlow(flow, &sessAgg)
	if err != nil {
		return err
	}

	//We only know how to stitch TCP and UDP
	//If the protocol is something out, write it out without stitching
	if proto != protocols.TCP &&
		proto != protocols.UDP {
		return writer.Write(&sessAgg)
	}

	//try to insert the new session aggregate.
	//overwrite and return the conflicting aggregate if an aggregate
	//already exists with the same AggregateQuery (Flow Key + Exporter)
	var oldSessAgg session.Aggregate
	info, err := sessionsColl.Find(&sessAgg.AggregateQuery).Apply(mgo.Change{
		Upsert: true,
		Update: &sessAgg,
	}, &oldSessAgg)

	if err != nil {
		return err
	}

	//If we overwrote an old agg, we need to decide whether we
	//should write out the old aggregate or if we should merge
	//our current aggregate in with it
	if info.Updated == 1 {

		flowStart, err := flow.FlowStartMilliseconds()
		if err != nil {
			return err
		}
		maxExpireTime := flowStart - s.sameSessionThreshold

		//grab the latest FlowEnd from the old session aggregate
		oldSessAggFlowEnd := oldSessAgg.FlowEndMillisecondsAB
		if oldSessAgg.FlowEndMillisecondsBA > oldSessAggFlowEnd {
			oldSessAggFlowEnd = oldSessAgg.FlowEndMillisecondsBA
		}

		//there is a good chance that the old session aggregate
		//just hasn't been flushed out yet.

		//If the old session happened within the same session threshold and
		//didn't end via a clean TCP teardown, then update the aggregate
		if oldSessAggFlowEnd >= maxExpireTime &&
			!(oldSessAgg.ProtocolIdentifier == protocols.TCP &&
				(srcMapping == session.ASource && oldSessAgg.FlowEndReasonAB == ipfix.EndOfFlow) ||
				(srcMapping == session.BSource && oldSessAgg.FlowEndReasonBA == ipfix.EndOfFlow)) {

			//merge the two aggregates
			err = sessAgg.Merge(&oldSessAgg)
			if err != nil {
				return err
			}

			err = sessionsColl.UpdateId(oldSessAgg.ID, &sessAgg)
			if err != nil {
				return err
			}
		} else {
			//The old connection is outside the same session threshold or
			//ended with a clean TCP teardown

			//write out the old session aggregate
			err = writer.Write(&oldSessAgg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
