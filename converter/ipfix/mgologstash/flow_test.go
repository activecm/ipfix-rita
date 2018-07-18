package mgologstash_test

import (
	"math"
	"testing"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/ipfix/mgologstash"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/stretchr/testify/require"
)

func getTestFlow1() *mgologstash.Flow {
	flow := &mgologstash.Flow{
		Host: "A",
	}
	flow.Netflow.SourceIPv4 = "1.1.1.1"
	flow.Netflow.SourcePort = 24846
	flow.Netflow.DestinationIPv4 = "2.2.2.2"
	flow.Netflow.DestinationPort = 53
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flow.Netflow.OctetTotalCount = int64(math.MaxUint32 + 1)
	flow.Netflow.PacketTotalCount = int64(math.MaxUint32 + 1)
	flow.Netflow.ProtocolIdentifier = protocols.UDP
	flow.Netflow.IPClassOfService = 3
	flow.Netflow.VlanID = 4
	flow.Netflow.FlowEndReason = ipfix.ActiveTimeout
	flow.Netflow.Version = 5
	return flow
}

func getTestFlow2() *mgologstash.Flow {
	flow := &mgologstash.Flow{
		Host: "B",
	}
	flow.Netflow.SourceIPv4 = "2.2.2.2"
	flow.Netflow.SourcePort = 53
	flow.Netflow.DestinationIPv4 = "1.1.1.1"
	flow.Netflow.DestinationPort = 24846
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:40:40.555Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:40:40.555Z"
	flow.Netflow.OctetTotalCount = int64(math.MaxUint32 - 1)
	flow.Netflow.PacketTotalCount = int64(math.MaxUint32 - 1)
	flow.Netflow.ProtocolIdentifier = protocols.TCP
	flow.Netflow.IPClassOfService = 30
	flow.Netflow.VlanID = 44
	flow.Netflow.FlowEndReason = ipfix.IdleTimeout
	flow.Netflow.Version = 54
	return flow
}

func getTestFlow3() *mgologstash.Flow {
	flow := &mgologstash.Flow{
		Host: "C",
	}
	flow.Netflow.SourceIPv4 = "3.3.3.3"
	flow.Netflow.SourcePort = 28972
	flow.Netflow.DestinationIPv4 = "4.4.4.4"
	flow.Netflow.DestinationPort = 443
	flow.Netflow.FlowStartMilliseconds = "2018-05-02T05:40:12.333Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-02T05:40:12.444Z"
	flow.Netflow.OctetTotalCount = math.MaxInt64
	flow.Netflow.PacketTotalCount = math.MaxInt64
	flow.Netflow.ProtocolIdentifier = protocols.TCP
	flow.Netflow.IPClassOfService = 50
	flow.Netflow.VlanID = 12
	flow.Netflow.FlowEndReason = ipfix.ActiveTimeout
	flow.Netflow.Version = 1
	return flow
}

func TestParseLogstashTime(t *testing.T) {
	flow := mgologstash.Flow{}
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flowStart, err := flow.FlowStartMilliseconds()
	require.Nil(t, err)
	flowEnd, err := flow.FlowEndMilliseconds()
	require.Nil(t, err)
	require.Equal(t, int64(1525473400*1000)+766, flowStart)
	require.Equal(t, int64(1525473400*1000)+960, flowEnd)
}

func TestV4V6Address(t *testing.T) {
	flow := mgologstash.Flow{}
	flow.Netflow.SourceIPv4 = "A"
	flow.Netflow.DestinationIPv4 = "B"
	require.Equal(t, flow.SourceIPAddress(), "A")
	require.Equal(t, flow.DestinationIPAddress(), "B")
	flow = mgologstash.Flow{}
	flow.Netflow.SourceIPv6 = "C"
	flow.Netflow.DestinationIPv6 = "D"
	require.Equal(t, flow.SourceIPAddress(), "C")
	require.Equal(t, flow.DestinationIPAddress(), "D")
}

func TestInheritance(t *testing.T) {
	var flow interface{} = &mgologstash.Flow{}
	_, ok := flow.(ipfix.Flow)
	require.True(t, ok)
}

//TODO: TestFillFromBSONMap
