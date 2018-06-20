package stitching

import (
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
)

//Stitcher implements the main worker logic for the convert command
type Stitcher struct {
	exporters            ExporterMap
	id                   int
	sameSessionThreshold uint64
}

//Stitch turns a Flow into a session... TODO
func (s Stitcher) Stitch(flow ipfix.Flow) error {
	proto := flow.ProtocolIdentifier()
	if proto == protocols.TCP && flow.PacketTotalCount() < 2 {
		return nil
	}
	if proto != protocols.TCP &&
		proto != protocols.UDP {
		//TODO zip with zeroes, write, and return
		return nil
	}

	//add the FlowEnd time of the last flow that could be merged with this flow
	flowStart, err := flow.FlowStartMilliseconds()
	if err != nil {
		return err
	}
	lastPossFlowEnd := flowStart - s.sameSessionThreshold

	exporter := s.exporters.Get(flow.Exporter())
	exporter.lastPossFlowEnds.Set(s.id, lastPossFlowEnd)
	//ensure the clock map is cleaned up when we exit
	defer exporter.lastPossFlowEnds.Clear(s.id)

	//TODO create session aggregate

	//TODO try to insert the session aggregate

	//TODO if the aggrgate already exists, expire it or merge it and update

	//check if this stitcher is working on the oldest flow
	minStitcherID := exporter.lastPossFlowEnds.FindMinID()
	if minStitcherID == s.id {
		//TODO clear out old session aggregates
	}

	return nil
}
