package data

import (
	"math"
	"testing"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/input/logstash/data/flowmap"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/require"
)

/*
//TestIFaceToInt64 will test sending three interfaces to the iFaceToInt64
//  function. The first value is an int64, the second is an int32 and the last
//  is a string (to test error handling)
func TestIFaceToInt64(t *testing.T) {
	tests := map[string]struct {
		faceVal     interface{}
		expectedVal int64
		err         error
	}{
		"int64IFace": {
			faceVal:     interface{}(int64(2147483648)),
			expectedVal: 2147483648,
			err:         nil,
		},
		"int32IFace": {
			faceVal:     interface{}(int(10)),
			expectedVal: 10,
			err:         nil,
		},
		"strIFace": {
			faceVal:     interface{}("Hello, World"),
			expectedVal: 0,
			err:         errors.New("could not convert Hello, World to int"),
		},
	}

	for name, test := range tests {
		t.Logf("Running test case: %s", name)

		testVal, err := iFaceToInt64(test.faceVal)

		assert.Equal(t, test.expectedVal, testVal, "iFaceToInt64 returned %d, should be %d", testVal, test.expectedVal)
		if err != nil {
			assert.NotEqual(t, test.err, nil, "iFaceToInt64 returned %s", test.err)
		}
	}
}
*/

func TestFillFromIPFIXBSONMap(t *testing.T) {
	var flow1 = new(Flow)
	var flowDeserializer = NewFlowDeserializer()
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

	var error1 = flowDeserializer.DeserializeNextMap(flowmap.NewFlowMapFromBSON(testData1), flow1)
	require.Nil(t, error1)

	require.Equal(t, testData1["_id"], flow1.ID)
	require.Equal(t, testData1["host"], flow1.Exporter())

	ipfixMap1 := (testData1["netflow"].(bson.M))
	require.Equal(t, ipfixMap1["sourceIPv4Address"], flow1.SourceIPAddress())
	require.Equal(t, ipfixMap1["postNATDestinationIPv4Address"], flow1.DestinationIPAddress())
	require.Equal(t, uint8(ipfixMap1["version"].(int)), flow1.Version())
	require.Equal(t, int64(ipfixMap1["packetTotalCount"].(int64)), flow1.PacketTotalCount())
	require.Equal(t, int64(ipfixMap1["octetTotalCount"].(int64)), flow1.OctetTotalCount())
	require.Equal(t, uint16(ipfixMap1["sourceTransportPort"].(int)), flow1.SourcePort())
	require.Equal(t, ipfixMap1["flowStartMilliseconds"], flow1.Netflow.FlowStartMilliseconds)
	require.Equal(t, protocols.Identifier(ipfixMap1["protocolIdentifier"].(int)), flow1.ProtocolIdentifier())
	require.Equal(t, uint16(ipfixMap1["postNAPTDestinationTransportPort"].(int)), flow1.DestinationPort())
	require.Equal(t, ipfixMap1["flowEndMilliseconds"], flow1.Netflow.FlowEndMilliseconds)
	require.Equal(t, input.FlowEndReason(ipfixMap1["flowEndReason"].(int)), flow1.FlowEndReason())

	var flow2 = new(Flow)
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

	var error2 = flowDeserializer.DeserializeNextMap(flowmap.NewFlowMapFromBSON(testData2), flow2)
	require.Nil(t, error2)
	require.Equal(t, "2001:db8:85a3:8d3:1319:8a2e:370:7348", flow2.DestinationIPAddress())
	require.Equal(t, uint16(57), flow2.DestinationPort())
}

func TestUptimeRelativeTimestamps(t *testing.T) {
	initTimeMap := bson.M{
		"_id": bson.ObjectId("5c06f8a7fe8088957d0000c5"),
		"netflow": bson.M{
			"version":                    10,
			"samplingPacketInterval":     1,
			"systemInitTimeMilliseconds": int64(1539907077250),
			"meteringProcessId":          92091,
			"selectorAlgorithm":          1,
			"samplingPacketSpace":        0,
		},
		"@timestamp": "\"2018-10-24T07:09:44.000Z\"",
		"@version":   "1",
		"host":       "172.22.0.1",
	}

	relativeTsFlow := bson.M{
		"_id": bson.ObjectId("5c06f8a7fe8088957d000097"),
		"netflow": bson.M{
			"sourceTransportPort":      53,
			"protocolIdentifier":       17,
			"egressInterface":          3,
			"packetDeltaCount":         1,
			"ingressInterface":         3,
			"flowStartSysUpTime":       457240831,
			"ipClassOfService":         0,
			"destinationTransportPort": 55539,
			"octetDeltaCount":          384,
			"version":                  10,
			"sourceIPv4Address":        "23.74.25.192",
			"tcpControlBits":           0,
			"ipVersion":                4,
			"destinationIPv4Address":   "10.55.200.10",
			"flowEndSysUpTime":         457240959,
			"icmpTypeCodeIPv4":         0,
			"vlanId":                   0,
		},
		"@timestamp": "\"2018-10-24T07:09:44.000Z\"",
		"@version":   "1",
		"host":       "172.22.0.1",
	}
	flow := Flow{}
	flowDeserializer := NewFlowDeserializer()

	var error1 = flowDeserializer.DeserializeNextMap(flowmap.NewFlowMapFromBSON(initTimeMap), &flow)

	//an error will be returned as flow should not have been filled.
	require.NotNil(t, error1)

	initTime, initTimeOk := flowDeserializer.ipfixExporterAbsUptimes["172.22.0.1"]
	require.True(t, initTimeOk)
	require.Equal(t, int64(1539907077250), initTime)

	var error2 = flowDeserializer.DeserializeNextMap(flowmap.NewFlowMapFromBSON(relativeTsFlow), &flow)
	require.Nil(t, error2)
	flowStart, timeErr := flow.FlowStartMilliseconds()
	require.Nil(t, timeErr)
	require.Equal(t, int64(1539907077250)+int64(457240831), flowStart)
	flowEnd, timeErr2 := flow.FlowEndMilliseconds()
	require.Nil(t, timeErr2)
	require.Equal(t, int64(1539907077250)+int64(457240959), flowEnd)
}

func TestFillFromNetflowv9BSONMap(t *testing.T) {
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
	flow := &Flow{}
	flowDeserializer := NewFlowDeserializer()

	err := flowDeserializer.DeserializeNextMap(flowmap.NewFlowMapFromBSON(inputMap), flow)
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

	flow2 := &Flow{}
	inputMap2 := bson.M{
		"_id":        bson.ObjectId("5b6b4e2e10a0cf244f0180aa"),
		"@timestamp": "\"2018-08-08T20:10:20.000Z\"",
		"host":       "73.149.157.171",
		"netflow": bson.M{
			"output_snmp":         1,
			"ipv6_src_addr":       "2002:db8:85a3:8d3:1319:8a2e:370:7345",
			"xlate_src_addr_ipv6": "2002:db8:85a3:8d3:1319:8a2e:370:7346",
			"ipv6_dst_addr":       "2002:db8:85a3:8d3:1319:8a2e:370:7347",
			"xlate_dst_addr_ipv6": "2001:db8:85a3:8d3:1319:8a2e:370:7348",
			"input_snmp":          1,
			"ipv4_next_hop":       "0.0.0.0",
			"version":             9,
			"flow_seq_num":        192,
			"flowset_id":          256,
			"in_pkts":             2,
			"in_bytes":            603,
			"l4_src_port":         47608,
			"first_switched":      "2018-08-08T20:10:21.000Z",
			"xlate_src_port":      47608,
			"protocol":            6,
			"xlate_dst_port":      444,
			"l4_dst_port":         443,
			"last_switched":       "2018-08-08T20:10:21.000Z",
		},
		"@version": "1",
	}

	var error2 = flowDeserializer.DeserializeNextMap(flowmap.NewFlowMapFromBSON(inputMap2), flow2)
	require.Nil(t, error2)
	require.Equal(t, "2001:db8:85a3:8d3:1319:8a2e:370:7348", flow2.DestinationIPAddress())
	require.Equal(t, uint16(444), flow2.DestinationPort())
}
