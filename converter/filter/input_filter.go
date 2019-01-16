package filter

import (
	"net"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/pkg/errors"
)

//FlowFilter returns whether a given flow matches a predicate or not
type FlowFilter interface {
	Match(input.Flow) (bool, error)
}

//NullFilter is a FlowFilter which doesn't match anything
type NullFilter struct {
}

//Match returns false, nil
func (n *NullFilter) Match(input.Flow) (bool, error) {
	return false, nil
}

//NewNullFilter returns a FlowFilter which doesn't match anything
func NewNullFilter() FlowFilter {
	return &NullFilter{}
}

//FlowBlacklist returns a match when a flow's src and dest IP addresses
//are both internal or both external addresses. Additionally, flows
//are matched when either the src or dest IP is on the NeverInclude list.
//Exceptions are made for flows on the AlwaysInclude list.
type FlowBlacklist struct {
	internal      []net.IPNet
	neverInclude  []net.IPNet
	alwaysInclude []net.IPNet
}

//NewFlowBlacklist returns a new FlowBlacklist. The resuling FlowBlacklist's
//Match method returns true when a flow's src and dest IP addresses
//are both internal or both external addresses. Additionally, Match returns
//true when either the src or dest IP is on the NeverInclude list.
//Exceptions are made for flows on the AlwaysInclude list.
func NewFlowBlacklist(internalNets, neverIncludeNets, alwaysIncludeNets []net.IPNet) FlowFilter {
	return &FlowBlacklist{
		internal:      internalNets,
		neverInclude:  neverIncludeNets,
		alwaysInclude: alwaysIncludeNets,
	}
}

//Match returns true when a flow's src and dest IP addresses
//are both internal or both external addresses. Additionally, flows
//are matched when either the src or dest IP is on the NeverInclude list.
//Exceptions are made for flows on the AlwaysInclude list.
func (f *FlowBlacklist) Match(flow input.Flow) (bool, error) {
	srcIP := net.ParseIP(flow.SourceIPAddress())
	if srcIP == nil {
		return false, errors.Errorf("failed to parse source IP address:\n%+v", flow)
	}
	destIP := net.ParseIP(flow.DestinationIPAddress())
	if destIP == nil {
		return false, errors.Errorf("failed to parse destination IP address:\n%+v", flow)
	}

	i := 0
	for i < len(f.alwaysInclude) {
		if f.alwaysInclude[i].Contains(srcIP) || f.alwaysInclude[i].Contains(destIP) {
			//Don't blacklist this flow if either the source or destination
			//are on the AlwaysInclude list.
			return false, nil
		}
		i++
	}

	i = 0
	for i < len(f.neverInclude) {
		if f.neverInclude[i].Contains(srcIP) || f.neverInclude[i].Contains(destIP) {
			//Blacklist any flow which has a source or destination matching the NeverInclude list
			return true, nil
		}
		i++
	}

	i = 0
	srcInternal := false
	for i < len(f.internal) && !srcInternal {
		srcInternal = f.internal[i].Contains(srcIP)
		i++
	}

	i = 0
	destInternal := false
	for i < len(f.internal) && !destInternal {
		destInternal = f.internal[i].Contains(destIP)
		i++
	}

	if srcInternal == destInternal {
		//Blacklist any flow which is internal -> interal or external -> external
		return true, nil
	}

	return false, nil
}
