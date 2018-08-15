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

func TestFillFromIPFIXBSONMapNATTranslation(t *testing.T) {
	var flow1 = new(mgologstash.Flow)
	var testData1 = bson.M{
		"_id":  bson.ObjectId("5b72d69af6a43336c6004e07"),
		"host": "A",
		"netflow": bson.M{
			"sourceIPv4Address":                "1.1.1.1",
			"sourceTransportPort":              24846,
			"destinationIPv4Address":           "2.2.2.2",
			"destinationIPv6Address":           "2002:db8:85a3:8d3:1319:8a2e:370:7348",
			"destinationTransportPort":         53,
			"flowStartMilliseconds":            "2018-05-04T22:36:40.766Z",
			"flowEndMilliseconds":              "2018-05-04T22:36:40.960Z",
			"octetTotalCount":                  int64(math.MaxUint32 + 1),
			"packetTotalCount":                 int64(math.MaxUint32 + 1),
			"protocolIdentifier":               int(protocols.UDP),
			"flowEndReason":                    int(input.ActiveTimeout),
			"version":                          10,
			"postNATDestinationIPv4Address":    "5.5.5.5",
			"postNAPTDestinationTransportPort": 55,
			"postNATDestinationIPv6Address":    "2001:db8:85a3:8d3:1319:8a2e:370:7348",
		},
	}

	var error1 = flow1.FillFromBSONMap(testData1)
	require.Nil(t, error1)
	require.Equal(t, flow1.DestinationIPAddress(), "5.5.5.5")
	require.Equal(t, flow1.DestinationPort(), uint16(55))

	var flow2 = new(mgologstash.Flow)
	var testData2 = bson.M{
		"_id":  bson.ObjectId("5b72d69af6a43336c6004e07"),
		"host": "A",
		"netflow": bson.M{
			"sourceIPv4Address":                "1.1.1.1",
			"sourceTransportPort":              24846,
			"destinationIPv6Address":           "2002:db8:85a3:8d3:1319:8a2e:370:7348",
			"destinationTransportPort":         53,
			"flowStartMilliseconds":            "2018-05-04T22:36:40.766Z",
			"flowEndMilliseconds":              "2018-05-04T22:36:40.960Z",
			"octetTotalCount":                  int64(math.MaxUint32 + 1),
			"packetTotalCount":                 int64(math.MaxUint32 + 1),
			"protocolIdentifier":               int(protocols.UDP),
			"flowEndReason":                    int(input.ActiveTimeout),
			"version":                          10,
			"postNAPTDestinationTransportPort": 57,
			"postNATDestinationIPv6Address":    "2001:db8:85a3:8d3:1319:8a2e:370:7348",
		},
	}

	var error2 = flow2.FillFromBSONMap(testData2)
	require.Nil(t, error2)
	require.Equal(t, flow2.DestinationIPAddress(), "2001:db8:85a3:8d3:1319:8a2e:370:7348")
	require.Equal(t, flow2.DestinationPort(), uint16(57))
}

func TestFillFromNetflowv9BSONMapIPv4(t *testing.T) {
	inputMap := bson.M{
		"_id":        bson.ObjectId("5b6b4e2e10a0cf244f0180aa"),
		"@timestamp": "\"2018-08-08T20:10:20.000Z\"",
		"host":       "73.149.157.171",
		"netflow": bson.M{
			"output_snmp":         1,
			"ipv4_src_addr":       "192.168.168.65",
			"xlate_dst_addr_ipv4": "192.168.168.168",
			"input_snmp":          1,
			"ipv4_next_hop":       "0.0.0.0",
			"version":             9,
			"flow_seq_num":        192,
			"flowset_id":          256,
			"in_pkts":             2,
			"in_bytes":            603,
			"ipv4_dst_addr":       "192.168.168.168",
			"xlate_src_addr_ipv4": "192.168.168.65",
			"l4_src_port":         47608,
			"first_switched":      "2018-08-08T20:10:21.000Z",
			"xlate_src_port":      47608,
			"protocol":            6,
			"xlate_dst_port":      443,
			"l4_dst_port":         443,
			"last_switched":       "2018-08-08T20:10:21.000Z",
		},
		"@version": "1",
	}
	flow := &mgologstash.Flow{}
	err := flow.FillFromBSONMap(inputMap)
	require.Nil(t, err)
	require.Equal(t, inputMap["_id"], flow.ID)
	require.Equal(t, inputMap["host"], flow.Exporter())

	netflowMap := (inputMap["netflow"].(bson.M))
	require.Equal(t, netflowMap["ipv4_src_addr"], flow.SourceIPAddress())
	require.Equal(t, netflowMap["xlate_dst_addr_ipv4"], flow.DestinationIPAddress())
	require.Equal(t, uint8(netflowMap["version"].(int)), flow.Version())
	require.Equal(t, int64(netflowMap["in_pkts"].(int)), flow.PacketTotalCount())
	require.Equal(t, int64(netflowMap["in_bytes"].(int)), flow.OctetTotalCount())
	require.Equal(t, uint16(netflowMap["l4_src_port"].(int)), flow.SourcePort())
	require.Equal(t, netflowMap["first_switched"], flow.Netflow.FlowStartMilliseconds)
	require.Equal(t, protocols.Identifier(netflowMap["protocol"].(int)), flow.ProtocolIdentifier())
	require.Equal(t, uint16(netflowMap["l4_dst_port"].(int)), flow.DestinationPort())
	require.Equal(t, netflowMap["last_switched"], flow.Netflow.FlowEndMilliseconds)
	//assume end of flow since we don't have the data
	require.Equal(t, input.EndOfFlow, flow.FlowEndReason())
}
