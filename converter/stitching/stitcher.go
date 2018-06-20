package stitching

import (
	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	mgo "gopkg.in/mgo.v2"
)

//Stitcher implements the main worker logic for the convert command
type Stitcher struct {
	environment.Environment
	exporters             ExporterMap
	writer                output.SessionWriter
	id                    int
	inactiveTimeoutMillis uint64
	sessionsColl          *mgo.Collection
}

//NewStitcher creates a new Stitching worker with the default
//session inactive timeout (1 minute)
func NewStitcher(env environment.Environment, exporters ExporterMap,
	writer output.SessionWriter, stitcherID int) Stitcher {

	return Stitcher{
		Environment: env,
		exporters:   exporters,
		writer:      writer,
		id:          stitcherID,
		inactiveTimeoutMillis: 60 * 60 * 1000,
		sessionsColl:          env.DB.NewSessionsConnection(),
	}
}

//Stitch turns a Flow into a session... TODO
func (s Stitcher) Stitch(flow ipfix.Flow) error {

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
		return s.writer.Write(&sessAgg)
	}

	//add the FlowEnd time of the last flow that could be merged with this flow
	//to the per exporter clock map
	flowStart, err := flow.FlowStartMilliseconds()
	if err != nil {
		return err
	}
	lastPossFlowEnd := flowStart - s.inactiveTimeoutMillis

	exporter := s.exporters.Get(flow.Exporter())
	exporter.lastPossFlowEnds.Set(s.id, lastPossFlowEnd)
	//ensure the clock map is cleaned up when we exit
	defer exporter.lastPossFlowEnds.Clear(s.id)

	//try to insert the new session aggregate
	//return the conflicting aggregate if an aggregate
	//already exists with the same AggregateQuery (Flow Key + Exporter)
	var oldSessAgg session.Aggregate
	_, err = s.sessionsColl.Find(&sessAgg.AggregateQuery).Apply(mgo.Change{
		Upsert: true,
		Update: &sessAgg,
	}, &oldSessAgg)

	//If err is a duplicate key error, oldSessAgg holds the existing aggregate
	if err != nil {
		mgoErr, ok := err.(*mgo.LastError)
		if !ok {
			return err
		}

		//see https://github.com/mongodb/mongo/blob/master/src/mongo/base/error_codes.err
		if mgoErr.Code != 11000 {
			return err
		}

		//grab the latest FlowEnd from the old session aggregate
		oldSessAggFlowEnd := oldSessAgg.FlowEndMillisecondsAB
		if oldSessAgg.FlowEndMillisecondsBA > oldSessAggFlowEnd {
			oldSessAggFlowEnd = oldSessAgg.FlowEndMillisecondsBA
		}

		//there is a chance that the old session aggregate
		//just hasn't been flushed out yet.

		//If the old session happened within the inactive timeout and
		//didn't end via a clean TCP teardown, then update the aggregate
		if oldSessAggFlowEnd >= lastPossFlowEnd &&
			!(oldSessAgg.ProtocolIdentifier == protocols.TCP &&
				(srcMapping == session.ASource && oldSessAgg.FlowEndReasonAB == ipfix.EndOfFlow) ||
				(srcMapping == session.BSource && oldSessAgg.FlowEndReasonBA == ipfix.EndOfFlow)) {

			//merge the two aggregates
			err := sessAgg.Merge(&oldSessAgg)
			if err != nil {
				return err
			}

			err = s.sessionsColl.UpdateId(oldSessAgg.ID, &sessAgg)
			if err != nil {
				return err
			}
		} else {
			//The old connection is outside the inactive timeout or
			//ended with a clean TCP teardown

			//set the stored session aggregate to our new aggregate
			err := s.sessionsColl.UpdateId(oldSessAgg.ID, &sessAgg)
			if err != nil {
				return err
			}

			//write out the old session aggregate
			err = s.writer.Write(&oldSessAgg)
			if err != nil {
				return err
			}
		}
	}

	//check if this stitcher is working on the oldest flow
	minStitcherID := exporter.lastPossFlowEnds.FindMinID()
	if minStitcherID == s.id {
		//TODO clear out old session aggregates
	}

	return nil
}
