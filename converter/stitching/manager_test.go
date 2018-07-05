package stitching_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/activecm/ipfix-rita/converter/environmenttest"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching"
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

var milliseconds = uint64(1000 * 60 * 60)

//newTestingStitchingManager is a helper for creating
//a stitching manager so tests don't get bogged down with setup code
func newTestingStitchingManager() stitching.Manager {
	sameSessionThreshold :=  milliseconds //milliseconds
	numStitchers := int32(5)                       //number of workers
	stitcherBufferSize := 5                        //number of flows that are buffered for each worker
	outputBufferSize := 5                          //number of session aggregates that are buffered for output

	return stitching.NewManager(
		sameSessionThreshold,
		numStitchers,
		stitcherBufferSize,
		outputBufferSize,
	)
}

func requireFlowStitchedWithZeroes(t *testing.T, flow ipfix.Flow, sess *session.Aggregate) {
	sourceIsA := flow.SourceIPAddress() < flow.DestinationIPAddress()
	if sourceIsA {
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

func requireFlowStitchedWithNoZeroes(t *testing.T, flow ipfix.Flow, sess *session.Aggregate) {
	// Todo : Needs to update based on No Zeroes understanding
	sourceIsA := flow.SourceIPAddress() < flow.DestinationIPAddress()
	if sourceIsA {
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

func TestTwoIcmpFlowSameSource(t *testing.T) {
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
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 100
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

func TestTwoIcmpFlowFlippedSource(t *testing.T){
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
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 100
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

	timeOut1 := sessions[0].FlowEndMillisecondsBA + milliseconds
	timeOut2 := sessions[1].FlowStartMillisecondsAB

	if timeOut2 >= timeOut1 {
		t.Fatalf("The difference between the flowEnd of the first connection and the flowStart of the next must be less than the sameSessionTimeout")
	}
}

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

func TestTwoUDPFlowSameSource(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
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
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 100
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	//ensure two aggregates are created since its UDP
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

	timeOut1 := sessions[0].FlowEndMillisecondsAB + milliseconds
	timeOut2 := sessions[1].FlowStartMillisecondsAB

	if timeOut2 >= timeOut1 {
		t.Fatalf("The difference between the flowEnd of the first connection and the flowStart of the next must be less than the sameSessionTimeout")
	}
}

func TestTwoUPDFlowFlippedSource(t *testing.T){
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
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
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 100
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	//ensure two aggregates are created since its UDP
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

	timeOut1 := sessions[0].FlowEndMillisecondsBA + milliseconds
	timeOut2 := sessions[1].FlowStartMillisecondsAB

	if timeOut2 >= timeOut1 {
		t.Fatalf("The difference between the flowEnd of the first connection and the flowStart of the next must be less than the sameSessionTimeout")
	}
}

func TestTwoUPDFlowOneAggregateZero(t *testing.T){
	//Todo : Needs to understand the condition to identify if its one aggreate only with two udp flow.
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	flow1 := ipfix.NewFlowMock()
	flow1.MockSourceIPAddress = "1.1.1.1"
	flow1.MockSourcePort = 0
	flow1.MockDestinationIPAddress = "2.2.2.2"
	flow1.MockDestinationPort = 771
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
	flow2.MockFlowStartMilliseconds = flow1.MockFlowEndMilliseconds + 100
	flow2.MockFlowEndMilliseconds = flow2.MockFlowStartMilliseconds + (flow1.MockFlowEndMilliseconds - flow1.MockFlowStartMilliseconds)

	stitchingManager := newTestingStitchingManager()
	sessions, errs := stitchingManager.RunSync([]ipfix.Flow{flow1, flow2}, env.DB)

	//ensure 1 aggregate is created since its UDP
	require.Len(t, sessions, 1)

	//ensure there were no errors
	if len(errs) != 0 {
		for i := range errs {
			t.Error(errs[i])
		}
	}
	require.Len(t, errs, 0)

	requireFlowStitchedWithZeroes(t, flow1, sessions[0])
	//requireFlowStitchedWithZeroes(t, flow2, sessions[1])

	timeOut1 := sessions[0].FlowEndMillisecondsBA + milliseconds
	timeOut2 := sessions[1].FlowStartMillisecondsAB

	if timeOut2 >= timeOut1 {
		t.Fatalf("The difference between the flowEnd of the first connection and the flowStart of the next must be less than the sameSessionTimeout")
	}
}