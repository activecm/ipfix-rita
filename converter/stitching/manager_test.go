package stitching

import (
	"runtime"
	"testing"
	"time"

	"github.com/activecm/ipfix-rita/converter/environmenttest"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/stretchr/testify/require"
)

/*
Logstash Flow Sample
{
	"_id" : ObjectId("5b371b96ca8fc40c3c001b24"),
	"@timestamp" : "\"2018-06-30T05:54:57.000Z\"",
	"host" : "172.21.0.1",
	"@version" : "1",
	"netflow" : {
		"flowEndReason" : 1,
		"destinationTransportPort" : 53,
		"sourceTransportPort" : 28752,
		"ipClassOfService" : 0,
		"flowStartMilliseconds" : "2018-05-04T22:36:10.493Z",
		"destinationIPv4Address" : "189.6.48.3",
		"sourceIPv4Address" : "104.131.28.214",
		"flowAttributes" : 0,
		"octetTotalCount" : 79,
		"flowEndMilliseconds" : "2018-05-04T22:36:10.618Z",
		"version" : 10,
		"vlanId" : 0,
		"protocolIdentifier" : 17,
		"packetTotalCount" : 1
	}
}

Protocol with identifier
{
	1:   ICMP
	6:   TCP
	17:  UDP
	58:  IPv6_ICMP
	132: SCTP
	142: ROHC
}

*/

/*  **********  Helper Variables  **********  */

var oneMinuteMillis = int64(1000 * 60)
var thirtySecondsMillis = int64(1000 * 30)

/*  **********  Helper Functions  **********  */

//newTestingStitchingManager is a helper for creating
//a stitching manager so tests don't get bogged down with setup code
func newTestingStitchingManager() Manager {
	sameSessionThreshold := oneMinuteMillis //milliseconds
	numStitchers := int32(5)                //number of workers
	stitcherBufferSize := 5                 //number of flows that are buffered for each worker
	outputBufferSize := 5                   //number of session aggregates that are buffered for output

	return NewManager(
		sameSessionThreshold,
		numStitchers,
		stitcherBufferSize,
		outputBufferSize,
	)
}

//requireFlowStitchedWithZeroes ensures a flow was assigned to only
//one side of a session aggregate. Additionally, this ensures
//no other flows were aggregated into the aggregate.
func requireFlowStitchedWithZeroes(t *testing.T, flow ipfix.Flow, sess *session.Aggregate) {
	sourceIsA := flow.SourceIPAddress() < flow.DestinationIPAddress()
	if sourceIsA {
		require.True(t, sess.FilledFromSourceA)
		require.False(t, sess.FilledFromSourceB)
		require.Equal(t, flow.Exporter(), sess.Exporter)
		require.Equal(t, flow.ProtocolIdentifier(), sess.ProtocolIdentifier)

		//ensure Source -> Dest information matches A -> B
		require.Equal(t, flow.SourceIPAddress(), sess.IPAddressA)
		require.Equal(t, flow.DestinationIPAddress(), sess.IPAddressB)
		require.Equal(t, flow.SourcePort(), sess.PortA)
		require.Equal(t, flow.DestinationPort(), sess.PortB)
		require.Equal(t, flow.OctetTotalCount(), sess.OctetTotalCountAB)
		require.Equal(t, flow.PacketTotalCount(), sess.PacketTotalCountAB)
		require.Equal(t, flow.FlowEndReason(), sess.FlowEndReasonAB)

		startTime, err := flow.FlowStartMilliseconds()
		require.Nil(t, err)
		endTime, err := flow.FlowEndMilliseconds()
		require.Nil(t, err)
		require.Equal(t, startTime, sess.FlowStartMillisecondsAB)
		require.Equal(t, endTime, sess.FlowEndMillisecondsAB)

		//ensure B -> A information is Zero/ Nil
		require.Zero(t, sess.OctetTotalCountBA)
		require.Zero(t, sess.PacketTotalCountBA)
		require.Zero(t, sess.FlowStartMillisecondsBA)
		require.Zero(t, sess.FlowEndMillisecondsBA)
		require.Equal(t, ipfix.Nil, sess.FlowEndReasonBA)
	} else {
		require.False(t, sess.FilledFromSourceA)
		require.True(t, sess.FilledFromSourceB)
		require.Equal(t, flow.Exporter(), sess.Exporter)
		require.Equal(t, flow.ProtocolIdentifier(), sess.ProtocolIdentifier)

		//ensure Source -> Dest information matches B -> A
		require.Equal(t, flow.SourceIPAddress(), sess.IPAddressB)
		require.Equal(t, flow.DestinationIPAddress(), sess.IPAddressA)
		require.Equal(t, flow.SourcePort(), sess.PortB)
		require.Equal(t, flow.DestinationPort(), sess.PortA)
		require.Equal(t, flow.OctetTotalCount(), sess.OctetTotalCountBA)
		require.Equal(t, flow.PacketTotalCount(), sess.PacketTotalCountBA)
		require.Equal(t, flow.FlowEndReason(), sess.FlowEndReasonBA)

		startTime, err := flow.FlowStartMilliseconds()
		require.Nil(t, err)
		endTime, err := flow.FlowEndMilliseconds()
		require.Nil(t, err)
		require.Equal(t, startTime, sess.FlowStartMillisecondsBA)
		require.Equal(t, endTime, sess.FlowEndMillisecondsBA)

		//ensure A -> B information is Zero/ Nil
		require.Zero(t, sess.OctetTotalCountAB)
		require.Zero(t, sess.PacketTotalCountAB)
		require.Zero(t, sess.FlowStartMillisecondsAB)
		require.Zero(t, sess.FlowEndMillisecondsAB)
		require.Equal(t, ipfix.Nil, sess.FlowEndReasonAB)
	}
}

//requireFlowsStitchedSameSide ensures two flows were stitched into
//the same side of a session aggregate and that the other side is
//filled with zeroes
func requireFlowsStitchedSameSide(t *testing.T, flow1, flow2 ipfix.Flow, sessAgg *session.Aggregate) {
	if flow1.SourceIPAddress() < flow1.DestinationIPAddress() {
		//assigned to AB side of the session aggregate
		//require the other flow to have the same assignment
		require.True(t, flow2.SourceIPAddress() < flow2.DestinationIPAddress())
		require.True(t, sessAgg.FilledFromSourceA)
		require.False(t, sessAgg.FilledFromSourceB)

		//data shared between flow1, flow2, and sessAgg
		require.Equal(t, flow1.Exporter(), sessAgg.Exporter)
		require.Equal(t, flow1.ProtocolIdentifier(), sessAgg.ProtocolIdentifier)
		require.Equal(t, flow1.SourceIPAddress(), sessAgg.IPAddressA)
		require.Equal(t, flow1.DestinationIPAddress(), sessAgg.IPAddressB)
		require.Equal(t, flow1.SourcePort(), sessAgg.PortA)
		require.Equal(t, flow1.DestinationPort(), sessAgg.PortB)

		//additive fields
		require.Equal(t, flow1.OctetTotalCount()+flow2.OctetTotalCount(), sessAgg.OctetTotalCountAB)
		require.Equal(t, flow1.PacketTotalCount()+flow2.PacketTotalCount(), sessAgg.PacketTotalCountAB)

		//comparitive fields
		flow1StartTime, err := flow1.FlowStartMilliseconds()
		require.Nil(t, err)
		flow1EndTime, err := flow1.FlowEndMilliseconds()
		require.Nil(t, err)

		flow2StartTime, err := flow2.FlowStartMilliseconds()
		require.Nil(t, err)
		flow2EndTime, err := flow2.FlowEndMilliseconds()
		require.Nil(t, err)

		if flow1StartTime <= flow2StartTime {
			require.Equal(t, flow1StartTime, sessAgg.FlowStartMillisecondsAB)
		} else {
			require.Equal(t, flow2StartTime, sessAgg.FlowStartMillisecondsAB)
		}

		if flow1EndTime >= flow2EndTime {
			require.Equal(t, flow1EndTime, sessAgg.FlowEndMillisecondsAB)
			require.Equal(t, flow1.FlowEndReason(), sessAgg.FlowEndReasonAB)
		} else {
			require.Equal(t, flow2EndTime, sessAgg.FlowEndMillisecondsAB)
			require.Equal(t, flow2.FlowEndReason(), sessAgg.FlowEndReasonAB)
		}

		//ensure B -> A information is Zero/ Nil
		require.Zero(t, sessAgg.OctetTotalCountBA)
		require.Zero(t, sessAgg.PacketTotalCountBA)
		require.Zero(t, sessAgg.FlowStartMillisecondsBA)
		require.Zero(t, sessAgg.FlowEndMillisecondsBA)
		require.Equal(t, ipfix.Nil, sessAgg.FlowEndReasonBA)

	} else {
		//assigned to BA side of the session aggregate
		//require the other flow to have the same assignment
		require.True(t, flow2.SourceIPAddress() >= flow2.DestinationIPAddress())
		require.False(t, sessAgg.FilledFromSourceA)
		require.True(t, sessAgg.FilledFromSourceB)

		//data shared between flow1, flow2, and sessAgg
		require.Equal(t, flow1.Exporter(), sessAgg.Exporter)
		require.Equal(t, flow1.ProtocolIdentifier(), sessAgg.ProtocolIdentifier)
		require.Equal(t, flow1.SourceIPAddress(), sessAgg.IPAddressB)
		require.Equal(t, flow1.DestinationIPAddress(), sessAgg.IPAddressA)
		require.Equal(t, flow1.SourcePort(), sessAgg.PortB)
		require.Equal(t, flow1.DestinationPort(), sessAgg.PortA)

		//additive fields
		require.Equal(t, flow1.OctetTotalCount()+flow2.OctetTotalCount(), sessAgg.OctetTotalCountBA)
		require.Equal(t, flow1.PacketTotalCount()+flow2.PacketTotalCount(), sessAgg.PacketTotalCountBA)

		//comparitive fields
		flow1StartTime, err := flow1.FlowStartMilliseconds()
		require.Nil(t, err)
		flow1EndTime, err := flow1.FlowEndMilliseconds()
		require.Nil(t, err)

		flow2StartTime, err := flow2.FlowStartMilliseconds()
		require.Nil(t, err)
		flow2EndTime, err := flow2.FlowEndMilliseconds()
		require.Nil(t, err)

		if flow1StartTime <= flow2StartTime {
			require.Equal(t, flow1StartTime, sessAgg.FlowStartMillisecondsBA)
		} else {
			require.Equal(t, flow2StartTime, sessAgg.FlowStartMillisecondsBA)
		}

		if flow1EndTime >= flow2EndTime {
			require.Equal(t, flow1EndTime, sessAgg.FlowEndMillisecondsBA)
			require.Equal(t, flow1.FlowEndReason(), sessAgg.FlowEndReasonBA)
		} else {
			require.Equal(t, flow2EndTime, sessAgg.FlowEndMillisecondsBA)
			require.Equal(t, flow2.FlowEndReason(), sessAgg.FlowEndReasonBA)
		}

		//ensure A -> B information is Zero/ Nil
		require.Zero(t, sessAgg.OctetTotalCountAB)
		require.Zero(t, sessAgg.PacketTotalCountAB)
		require.Zero(t, sessAgg.FlowStartMillisecondsAB)
		require.Zero(t, sessAgg.FlowEndMillisecondsAB)
		require.Equal(t, ipfix.Nil, sessAgg.FlowEndReasonAB)
	}
}

//requireFlowsStitchedFlippedSides ensures two flows were stitched into
//opposite sides of a session aggregate
func requireFlowsStitchedFlippedSides(t *testing.T, flow1, flow2 ipfix.Flow, sessAgg *session.Aggregate) {
	require.True(t, sessAgg.FilledFromSourceA)
	require.True(t, sessAgg.FilledFromSourceB)
	flow1SourceIsA := flow1.SourceIPAddress() < flow1.DestinationIPAddress()
	if flow1SourceIsA {
		//require the otherflow is assigned to the other side
		require.True(t, flow2.SourceIPAddress() >= flow2.DestinationIPAddress())

		require.Equal(t, flow1.Exporter(), sessAgg.Exporter)
		require.Equal(t, flow1.ProtocolIdentifier(), sessAgg.ProtocolIdentifier)

		//ensure Flow1 Source -> Dest information matches A -> B
		require.Equal(t, flow1.SourceIPAddress(), sessAgg.IPAddressA)
		require.Equal(t, flow1.DestinationIPAddress(), sessAgg.IPAddressB)
		require.Equal(t, flow1.SourcePort(), sessAgg.PortA)
		require.Equal(t, flow1.DestinationPort(), sessAgg.PortB)

		require.Equal(t, flow1.OctetTotalCount(), sessAgg.OctetTotalCountAB)
		require.Equal(t, flow1.PacketTotalCount(), sessAgg.PacketTotalCountAB)
		require.Equal(t, flow1.FlowEndReason(), sessAgg.FlowEndReasonAB)

		flow1StartTime, err := flow1.FlowStartMilliseconds()
		require.Nil(t, err)
		flow1EndTime, err := flow1.FlowEndMilliseconds()
		require.Nil(t, err)
		require.Equal(t, flow1StartTime, sessAgg.FlowStartMillisecondsAB)
		require.Equal(t, flow1EndTime, sessAgg.FlowEndMillisecondsAB)

		//ensure Flow2 Source -> Dest information matches B -> A

		require.Equal(t, flow2.SourceIPAddress(), sessAgg.IPAddressB)
		require.Equal(t, flow2.DestinationIPAddress(), sessAgg.IPAddressA)
		require.Equal(t, flow2.SourcePort(), sessAgg.PortB)
		require.Equal(t, flow2.DestinationPort(), sessAgg.PortA)

		require.Equal(t, flow2.OctetTotalCount(), sessAgg.OctetTotalCountBA)
		require.Equal(t, flow2.PacketTotalCount(), sessAgg.PacketTotalCountBA)
		require.Equal(t, flow2.FlowEndReason(), sessAgg.FlowEndReasonBA)

		flow2StartTime, err := flow2.FlowStartMilliseconds()
		require.Nil(t, err)
		flow2EndTime, err := flow2.FlowEndMilliseconds()
		require.Nil(t, err)
		require.Equal(t, flow2StartTime, sessAgg.FlowStartMillisecondsBA)
		require.Equal(t, flow2EndTime, sessAgg.FlowEndMillisecondsBA)
	} else {
		//require the otherflow is assigned to the other side
		require.True(t, flow2.SourceIPAddress() < flow2.DestinationIPAddress())

		require.Equal(t, flow1.Exporter(), sessAgg.Exporter)
		require.Equal(t, flow1.ProtocolIdentifier(), sessAgg.ProtocolIdentifier)

		//ensure Flow1 Source -> Dest information matches B -> A
		require.Equal(t, flow1.SourceIPAddress(), sessAgg.IPAddressB)
		require.Equal(t, flow1.DestinationIPAddress(), sessAgg.IPAddressA)
		require.Equal(t, flow1.SourcePort(), sessAgg.PortB)
		require.Equal(t, flow1.DestinationPort(), sessAgg.PortA)

		require.Equal(t, flow1.OctetTotalCount(), sessAgg.OctetTotalCountBA)
		require.Equal(t, flow1.PacketTotalCount(), sessAgg.PacketTotalCountBA)
		require.Equal(t, flow1.FlowEndReason(), sessAgg.FlowEndReasonBA)

		flow1StartTime, err := flow1.FlowStartMilliseconds()
		require.Nil(t, err)
		flow1EndTime, err := flow1.FlowEndMilliseconds()
		require.Nil(t, err)
		require.Equal(t, flow1StartTime, sessAgg.FlowStartMillisecondsBA)
		require.Equal(t, flow1EndTime, sessAgg.FlowEndMillisecondsBA)

		//ensure Flow2 Source -> Dest information matches A -> B

		require.Equal(t, flow2.SourceIPAddress(), sessAgg.IPAddressA)
		require.Equal(t, flow2.DestinationIPAddress(), sessAgg.IPAddressB)
		require.Equal(t, flow2.SourcePort(), sessAgg.PortA)
		require.Equal(t, flow2.DestinationPort(), sessAgg.PortB)

		require.Equal(t, flow2.OctetTotalCount(), sessAgg.OctetTotalCountAB)
		require.Equal(t, flow2.PacketTotalCount(), sessAgg.PacketTotalCountAB)
		require.Equal(t, flow2.FlowEndReason(), sessAgg.FlowEndReasonAB)

		flow2StartTime, err := flow2.FlowStartMilliseconds()
		require.Nil(t, err)
		flow2EndTime, err := flow2.FlowEndMilliseconds()
		require.Nil(t, err)
		require.Equal(t, flow2StartTime, sessAgg.FlowStartMillisecondsAB)
		require.Equal(t, flow2EndTime, sessAgg.FlowEndMillisecondsAB)
	}
}

/*  **********  SelectStitcher Tests  **********  */
func TestSelectStitcherFairness(t *testing.T) {
	manager := newTestingStitchingManager()

	//generate a bunch of data
	//act like were distributing the data to the workers
	//ensure the data is split roughly evenly
	binCount := make(map[int]int)
	for i := 0; i < 1000; i++ {
		binCount[manager.selectStitcher(ipfix.NewFlowMock())]++
	}
	expected := 1000 / 5
	delta := 25
	for binNumber := range binCount {
		diff := binCount[binNumber] - expected
		if diff < 0 {
			diff *= -1
		}
		require.True(t, diff < delta)
	}
}

func TestSelectStitcherReproducible(t *testing.T) {
	manager := newTestingStitchingManager()

	flow1 := ipfix.NewFlowMock()
	flow2 := ipfix.NewFlowMock()

	//ensure the selectStitcher gives the same results
	//if the flow key is the same
	assignment1 := manager.selectStitcher(flow1)
	assignment2 := manager.selectStitcher(flow2)
	for i := 0; i < 100; i++ {
		newFlow := ipfix.NewFlowMock()

		newFlow.MockSourceIPAddress = flow1.SourceIPAddress()
		newFlow.MockDestinationIPAddress = flow1.DestinationIPAddress()
		newFlow.MockSourcePort = flow1.SourcePort()
		newFlow.MockDestinationPort = flow1.DestinationPort()
		newFlow.MockProtocolIdentifier = flow1.ProtocolIdentifier()
		newFlow.MockExporter = flow1.Exporter()
		require.Equal(t, assignment1, manager.selectStitcher(flow1))

		newFlow.MockSourceIPAddress = flow2.SourceIPAddress()
		newFlow.MockDestinationIPAddress = flow2.DestinationIPAddress()
		newFlow.MockSourcePort = flow2.SourcePort()
		newFlow.MockDestinationPort = flow2.DestinationPort()
		newFlow.MockProtocolIdentifier = flow2.ProtocolIdentifier()
		newFlow.MockExporter = flow2.Exporter()
		require.Equal(t, assignment2, manager.selectStitcher(flow2))
	}
}

func TestSelectStitcherFlippedFlowKeys(t *testing.T) {
	manager := newTestingStitchingManager()

	//repeat the test a few times since the data is random
	for i := 0; i < 100; i++ {
		flow1 := ipfix.NewFlowMock()
		assignment1 := manager.selectStitcher(flow1)

		//create a flow with a matching, flipped flow key
		flow2 := ipfix.NewFlowMock()
		flow2.MockSourceIPAddress = flow1.DestinationIPAddress()
		flow2.MockDestinationIPAddress = flow1.SourceIPAddress()
		flow2.MockSourcePort = flow1.DestinationPort()
		flow2.MockDestinationPort = flow1.SourcePort()
		flow2.MockProtocolIdentifier = flow1.ProtocolIdentifier()
		flow2.MockExporter = flow1.Exporter()

		require.Equal(t, assignment1, manager.selectStitcher(flow2))
	}
}

/*  **********  Stitching Manager Implementation Tests  **********  */
func TestGoRoutineLeaks(t *testing.T) {
	numGoRoutines := runtime.NumGoroutine()

	//Set up for an integration test
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	stitchingManager := newTestingStitchingManager()
	_, errs := stitchingManager.RunSync(
		[]ipfix.Flow{
			ipfix.NewFlowMock(),
			ipfix.NewFlowMock(),
		},
		env.DB,
	)
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)
	cleanup()

	//annoyingly, mgo may stay open for 15 seconds
	//see: gopkg.in/mgo.v2/server.go:301
	time.Sleep(15 * time.Second)
	require.Equal(t, numGoRoutines, runtime.NumGoroutine())
}

/*  **********  Stitching Manager ICMP Tests  **********  */

func TestSingleIcmpFlow(t *testing.T) {
	//Set up for an integration test
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	//Create the input flow from random data
	flow1 := ipfix.NewFlowMock()
	//Ensure the source comes before the destination alphabetically
	//to ensure the source is mapped to host "A", and the destination is
	//mapped to host "B"
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockDestinationIPAddress = "2.2.2.2"
	//Set the protocol to ICMP
	flow1.MockProtocolIdentifier = protocols.ICMP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	//run the stitching manager
	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1}, env.DB)

	//ensure only one aggregate is created
	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
}

func TestTwoICMPFlowsSameSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
	flow1.MockProtocolIdentifier = protocols.ICMP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be less than sameSessionTimeout
	//in order for this test be considered "inTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	//ensure two aggregates are created since its ICMP
	require.Len(t, sessions, 2)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestTwoICMPFlowsSameSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
	flow1.MockProtocolIdentifier = protocols.ICMP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be after the `sameSessionTimeout` has elapsed.
	//in order for this test be considered "OutOfTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	//ensure two aggregates are created since its ICMP
	require.Len(t, sessions, 2)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestTwoICMPFlowsFlippedSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
	flow1.MockProtocolIdentifier = protocols.ICMP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be less than sameSessionTimeout
	//in order for this test be considered "inTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	//ensure two aggregates are created since its ICMP
	require.Len(t, sessions, 2)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestTwoICMPFlowsFlippedSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
	flow1.MockProtocolIdentifier = protocols.ICMP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be after the `sameSessionTimeout` has elapsed.
	//in order for this test be considered "OutOfTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	//ensure two aggregates are created since its ICMP
	require.Len(t, sessions, 2)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

/*  **********  Stitching Manager UDP Tests  **********  */
func TestSingleUDPFlow(t *testing.T) {
	//Set up for an integration test
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	//Create the input flow from random data
	flow1 := ipfix.NewFlowMock()
	//Ensure the source comes before the destination alphabetically
	//to ensure the source is mapped to host "A", and the destination is
	//mapped to host "B"
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockDestinationIPAddress = "2.2.2.2"
	//Set the protocol to UDP
	flow1.MockProtocolIdentifier = protocols.UDP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	//run the stitching manager
	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1}, env.DB)

	//ensure only one aggregate is created
	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
}

func TestTwoUDPFlowSameSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.UDP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowsStitchedSameSide(t, flow1, flow2, sessions[0])
}

func TestTwoUDPFlowsSameSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.UDP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 2)

	//ensure the session timing mismatch error fires as a warning
	require.Len(t, errs, 1)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestTwoUDPFlowsFlippedSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.UDP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be less than sameSessionTimeout
	//in order for this test be considered "inTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowsStitchedFlippedSides(t, flow1, flow2, sessions[0])
}

func TestTwoUDPFlowsFlippedSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.UDP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be after the `sameSessionTimeout` has elapsed.
	//in order for this test be considered "OutOfTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 2)

	//ensure the session timing mismatch error fires as a warning
	require.Len(t, errs, 1)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestSingleTCPIdleOutFlow(t *testing.T) {
	//Set up for an integration test
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	//Create the input flow from random data
	flow1 := ipfix.NewFlowMock()
	//Ensure the source comes before the destination alphabetically
	//to ensure the source is mapped to host "A", and the destination is
	//mapped to host "B"
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockDestinationIPAddress = "2.2.2.2"
	//Set the protocol to TCP
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	//run the stitching manager
	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1}, env.DB)

	//ensure only one aggregate is created
	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
}

func TestTwoTCPIdleOutFlowsSameSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowsStitchedSameSide(t, flow1, flow2, sessions[0])
}

func TwoTCPIdleOutFlowsSameSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 2)

	require.Len(t, errs, 1)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
 	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestTwoTCPIdleOutFlowsFlippedSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be less than sameSessionTimeout
	//in order for this test be considered "inTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowsStitchedFlippedSides(t, flow1, flow2, sessions[0])
}

func TestTwoTCPIdleOutFlowsFlippedSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.IdleTimeout

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
  	//flow start of the next must be after the `sameSessionTimeout` has elapsed.
  	//in order for this test be considered "OutOfTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 2)

	require.Len(t, errs, 1)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestSingleTCPEOFFlow(t *testing.T) {
	//Set up for an integration test
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	//Create the input flow from random data
	flow1 := ipfix.NewFlowMock()
	//Ensure the source comes before the destination alphabetically
	//to ensure the source is mapped to host "A", and the destination is
	//mapped to host "B"
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockDestinationIPAddress = "2.2.2.2"
	//Set the protocol to TCP
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.EndOfFlow

	//run the stitching manager
	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1}, env.DB)

	//ensure only one aggregate is created
	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
}

func TestTwoTCPEOFFlowsSameSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.EndOfFlow

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 2)

	require.Len(t, errs, 1)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestTwoTCPEOFFlowsSameSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 29445
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 53
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.EndOfFlow

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockSourceIPAddress
	flow2.MockDestinationIPAddress = flow1.MockDestinationIPAddress
	flow2.MockSourcePort = flow1.MockSourcePort
	flow2.MockDestinationPort = flow1.MockDestinationPort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 2)

	require.Len(t, errs, 1)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}

func TestTwoTCPEOFFlowsFlippedSourceInTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.EndOfFlow

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be less than sameSessionTimeout
	//in order for this test be considered "inTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowsStitchedFlippedSides(t, flow1, flow2, sessions[0])
}

func TestTwoTCPEOFFlowsFlippedSourceOutOfTimeout(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
	flow1.MockProtocolIdentifier = protocols.TCP
	flow1.MockFlowEndReason = ipfix.EndOfFlow

	flow2 := ipfix.NewFlowMock()
	flow2.MockSourceIPAddress = flow1.MockDestinationIPAddress
	flow2.MockDestinationIPAddress = flow1.MockSourceIPAddress
	flow2.MockSourcePort = flow1.MockDestinationPort
	flow2.MockDestinationPort = flow1.MockSourcePort
	flow2.MockExporter = flow1.MockExporter
	flow2.MockProtocolIdentifier = flow1.MockProtocolIdentifier
	flow2.MockFlowEndReason = flow1.MockFlowEndReason

	//The difference between the flowEnd of the first connection and the
	//flow start of the next must be after the `sameSessionTimeout` has elapsed.
	//in order for this test be considered "OutOfTimeout"
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 5*thirtySecondsMillis
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	require.Len(t, sessions, 2)

	require.Len(t, errs, 1)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	requireFlowStitchedWithZeroes(t, flow2, sessions[1])
}