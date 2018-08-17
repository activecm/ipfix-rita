package mgologstash_test

import (
	"math"
	"testing"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/input/mgologstash"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
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
	flow.Netflow.FlowEndReason = input.ActiveTimeout
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
	flow.Netflow.FlowEndReason = input.IdleTimeout
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
	flow.Netflow.FlowEndReason = input.ActiveTimeout
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
	_, ok := flow.(input.Flow)
	require.True(t, ok)
}

func TestFillFromBSONMap(t *testing.T){
	var flow1 = new(mgologstash.Flow)
	var testData1 = bson.M{
		"_id": bson.ObjectId("5b72d69af6a43336c6004e07"),
		"host": "A",
		"netflow": bson.M {
		"sourceIPv4Address": "1.1.1.1",
		"sourceTransportPort" : 24846,
		"destinationIPv4Address" : "2.2.2.2",
		"destinationIPv6Address" : "2002:db8:85a3:8d3:1319:8a2e:370:7348",
		"destinationTransportPort" : 53,
		"flowStartMilliseconds" : "2018-05-04T22:36:40.766Z",
		"flowEndMilliseconds" : "2018-05-04T22:36:40.960Z",
		"octetTotalCount" : int64(math.MaxUint32 + 1),
		"packetTotalCount" : int64(math.MaxUint32 + 1),
		"protocolIdentifier" : int(protocols.UDP),
		"flowEndReason" : int(input.ActiveTimeout),
		"version" : 5,
		"postNATDestinationIPv4Address" : "5.5.5.5",
		"postNAPTDestinationTransportPort" : 55,
		"postNATDestinationIPv6Address" : "2001:db8:85a3:8d3:1319:8a2e:370:7348",
		},
	}

	var error1 = flow1.FillFromBSONMap(testData1);
	require.Nil(t, error1)
	require.Equal(t, flow1.DestinationIPAddress(), "5.5.5.5")
	require.Equal(t, flow1.DestinationPort(), uint16(55))

	var flow2 = new(mgologstash.Flow)
	var testData2 = bson.M{
		"_id": bson.ObjectId("5b72d69af6a43336c6004e07"),
		"host": "A",
		"netflow": bson.M {
		"sourceIPv4Address": "1.1.1.1",
		"sourceTransportPort" : 24846,
		"destinationIPv6Address" : "2002:db8:85a3:8d3:1319:8a2e:370:7348",
		"destinationTransportPort" : 53,
		"flowStartMilliseconds" : "2018-05-04T22:36:40.766Z",
		"flowEndMilliseconds" : "2018-05-04T22:36:40.960Z",
		"octetTotalCount" : int64(math.MaxUint32 + 1),
		"packetTotalCount" : int64(math.MaxUint32 + 1),
		"protocolIdentifier" : int(protocols.UDP),
		"flowEndReason" : int(input.ActiveTimeout),
		"version" : 5,
		"postNAPTDestinationTransportPort" : 57,
		"postNATDestinationIPv6Address" : "2001:db8:85a3:8d3:1319:8a2e:370:7348",
		},
	}

	var error2 = flow2.FillFromBSONMap(testData2);
	require.Nil(t, error2)
	require.Equal(t, flow2.DestinationIPAddress(), "2001:db8:85a3:8d3:1319:8a2e:370:7348")
	require.Equal(t, flow2.DestinationPort(), uint16(57))
}