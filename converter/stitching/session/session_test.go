package session_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestFromFlowASource(t *testing.T) {
	var sess session.Aggregate
	testFlow := ipfix.NewFlowMock()
	testFlow.MockSourceIPAddress = "1.1.1.1"
	testFlow.MockDestinationIPAddress = "2.2.2.2"
	session.FromFlow(testFlow, &sess)
	require.Equal(t, testFlow.SourceIPAddress(), sess.IPAddressA)
	require.Equal(t, testFlow.SourcePort(), sess.PortA)
	require.Equal(t, testFlow.DestinationIPAddress(), sess.IPAddressB)
	require.Equal(t, testFlow.DestinationPort(), sess.PortB)
	require.Equal(t, testFlow.Exporter(), sess.Exporter)
	require.Equal(t, testFlow.ProtocolIdentifier(), sess.ProtocolIdentifier)
	require.Equal(t, testFlow.MockFlowStartMilliseconds, sess.FlowStartMillisecondsAB)
	require.Equal(t, testFlow.MockFlowEndMilliseconds, sess.FlowEndMillisecondsAB)
	require.Equal(t, testFlow.OctetTotalCount(), sess.OctetTotalCountAB)
	require.Equal(t, testFlow.PacketTotalCount(), sess.PacketTotalCountAB)
	require.Equal(t, testFlow.FlowEndReason(), sess.FlowEndReasonAB)
}

func TestFromFlowADest(t *testing.T) {
	var sess session.Aggregate
	testFlow := ipfix.NewFlowMock()
	testFlow.MockSourceIPAddress = "2.2.2.2"
	testFlow.MockDestinationIPAddress = "1.1.1.1"
	session.FromFlow(testFlow, &sess)
	require.Equal(t, testFlow.SourceIPAddress(), sess.IPAddressB)
	require.Equal(t, testFlow.SourcePort(), sess.PortB)
	require.Equal(t, testFlow.DestinationIPAddress(), sess.IPAddressA)
	require.Equal(t, testFlow.DestinationPort(), sess.PortA)
	require.Equal(t, testFlow.Exporter(), sess.Exporter)
	require.Equal(t, testFlow.ProtocolIdentifier(), sess.ProtocolIdentifier)
	require.Equal(t, testFlow.MockFlowStartMilliseconds, sess.FlowStartMillisecondsBA)
	require.Equal(t, testFlow.MockFlowEndMilliseconds, sess.FlowEndMillisecondsBA)
	require.Equal(t, testFlow.OctetTotalCount(), sess.OctetTotalCountBA)
	require.Equal(t, testFlow.PacketTotalCount(), sess.PacketTotalCountBA)
	require.Equal(t, testFlow.FlowEndReason(), sess.FlowEndReasonBA)
}

func TestClear(t *testing.T) {
	var sess session.Aggregate
	testFlow := ipfix.NewFlowMock()
	session.FromFlow(testFlow, &sess)

	//ensure there is data
	require.Equal(t, testFlow.Exporter(), sess.Exporter)

	sess.Clear()
	require.Equal(t, bson.ObjectId(""), sess.ID)
	require.Equal(t, "", sess.IPAddressA)
	require.Equal(t, "", sess.IPAddressB)
	require.Equal(t, uint16(0), sess.PortA)
	require.Equal(t, uint16(0), sess.PortB)
	require.Equal(t, protocols.Identifier(0), sess.ProtocolIdentifier)
	require.Equal(t, "", sess.Exporter)
	require.Equal(t, uint64(0), sess.FlowStartMillisecondsAB)
	require.Equal(t, uint64(0), sess.FlowStartMillisecondsBA)
	require.Equal(t, uint64(0), sess.FlowEndMillisecondsAB)
	require.Equal(t, uint64(0), sess.FlowEndMillisecondsBA)
	require.Equal(t, uint64(0), sess.OctetTotalCountAB)
	require.Equal(t, uint64(0), sess.OctetTotalCountBA)
	require.Equal(t, uint64(0), sess.PacketTotalCountAB)
	require.Equal(t, uint64(0), sess.PacketTotalCountBA)
	require.Equal(t, ipfix.Nil, sess.FlowEndReasonAB)
	require.Equal(t, ipfix.Nil, sess.FlowEndReasonBA)
}

func TestMergeWrongFlowKeys(t *testing.T) {
	testFlowA := ipfix.NewFlowMock()
	testFlowB := ipfix.NewFlowMock()

	testFlowA.MockSourceIPAddress = "2.1.1.1"
	testFlowB.MockSourceIPAddress = "1.1.1.1"
	var sessA session.Aggregate
	var sessB session.Aggregate
	session.FromFlow(testFlowA, &sessA)
	session.FromFlow(testFlowB, &sessB)
	err := sessA.Merge(&sessB)

	require.NotNil(t, err)
}

func TestMergeSameDirectionSequential(t *testing.T) {
	testFlowA := ipfix.NewFlowMock()
	testFlowB := ipfix.NewFlowMock()

	testFlowA.MockSourceIPAddress = "1.1.1.1"
	testFlowB.MockSourceIPAddress = "1.1.1.1"

	testFlowA.MockSourcePort = 30000
	testFlowB.MockSourcePort = 30000

	testFlowA.MockDestinationIPAddress = "2.2.2.2"
	testFlowB.MockDestinationIPAddress = "2.2.2.2"

	testFlowA.MockDestinationPort = 4444
	testFlowB.MockDestinationPort = 4444

	testFlowA.MockProtocolIdentifier = protocols.UDP
	testFlowB.MockProtocolIdentifier = protocols.UDP

	testFlowB.MockExporter = testFlowA.Exporter()

	testFlowA.MockFlowStartMilliseconds = 100
	testFlowA.MockFlowEndMilliseconds = 200

	testFlowB.MockFlowStartMilliseconds = 300
	testFlowB.MockFlowEndMilliseconds = 400

	var sessA session.Aggregate
	var sessB session.Aggregate
	session.FromFlow(testFlowA, &sessA)
	session.FromFlow(testFlowB, &sessB)
	err := sessA.Merge(&sessB)

	require.Nil(t, err)

	//Don't mess with the flow key
	require.Equal(t, testFlowA.SourceIPAddress(), sessA.IPAddressA)
	require.Equal(t, testFlowA.SourcePort(), sessA.PortA)
	require.Equal(t, testFlowA.DestinationIPAddress(), sessA.IPAddressB)
	require.Equal(t, testFlowA.DestinationPort(), sessA.PortB)
	require.Equal(t, testFlowA.ProtocolIdentifier(), sessA.ProtocolIdentifier)
	require.Equal(t, testFlowA.Exporter(), sessA.Exporter)

	require.Equal(t, testFlowA.MockFlowStartMilliseconds, sessA.FlowStartMillisecondsAB)
	require.Equal(t, testFlowB.MockFlowEndMilliseconds, sessA.FlowEndMillisecondsAB)
	require.Equal(t, testFlowA.OctetTotalCount()+testFlowB.OctetTotalCount(), sessA.OctetTotalCountAB)
	require.Equal(t, testFlowA.PacketTotalCount()+testFlowB.PacketTotalCount(), sessA.PacketTotalCountAB)
	require.Equal(t, testFlowB.FlowEndReason(), sessA.FlowEndReasonAB)
}

func TestMergeSameDirectionAntiSequential(t *testing.T) {
	testFlowA := ipfix.NewFlowMock()
	testFlowB := ipfix.NewFlowMock()

	testFlowA.MockSourceIPAddress = "1.1.1.1"
	testFlowB.MockSourceIPAddress = "1.1.1.1"

	testFlowA.MockSourcePort = 30000
	testFlowB.MockSourcePort = 30000

	testFlowA.MockDestinationIPAddress = "2.2.2.2"
	testFlowB.MockDestinationIPAddress = "2.2.2.2"

	testFlowA.MockDestinationPort = 4444
	testFlowB.MockDestinationPort = 4444

	testFlowA.MockProtocolIdentifier = protocols.UDP
	testFlowB.MockProtocolIdentifier = protocols.UDP

	testFlowB.MockExporter = testFlowA.Exporter()

	testFlowA.MockFlowStartMilliseconds = 300
	testFlowA.MockFlowEndMilliseconds = 400

	testFlowB.MockFlowStartMilliseconds = 100
	testFlowB.MockFlowEndMilliseconds = 200

	var sessA session.Aggregate
	var sessB session.Aggregate
	session.FromFlow(testFlowA, &sessA)
	session.FromFlow(testFlowB, &sessB)
	err := sessA.Merge(&sessB)

	require.Nil(t, err)

	//Don't mess with the flow key
	require.Equal(t, testFlowA.SourceIPAddress(), sessA.IPAddressA)
	require.Equal(t, testFlowA.SourcePort(), sessA.PortA)
	require.Equal(t, testFlowA.DestinationIPAddress(), sessA.IPAddressB)
	require.Equal(t, testFlowA.DestinationPort(), sessA.PortB)
	require.Equal(t, testFlowA.ProtocolIdentifier(), sessA.ProtocolIdentifier)
	require.Equal(t, testFlowA.Exporter(), sessA.Exporter)

	require.Equal(t, testFlowB.MockFlowStartMilliseconds, sessA.FlowStartMillisecondsAB)
	require.Equal(t, testFlowA.MockFlowEndMilliseconds, sessA.FlowEndMillisecondsAB)
	require.Equal(t, testFlowA.OctetTotalCount()+testFlowB.OctetTotalCount(), sessA.OctetTotalCountAB)
	require.Equal(t, testFlowA.PacketTotalCount()+testFlowB.PacketTotalCount(), sessA.PacketTotalCountAB)
	require.Equal(t, testFlowA.FlowEndReason(), sessA.FlowEndReasonAB)
}

func TestMergeOppositeDirectionSequential(t *testing.T) {
	testFlowA := ipfix.NewFlowMock()
	testFlowB := ipfix.NewFlowMock()

	testFlowA.MockSourceIPAddress = "1.1.1.1"
	testFlowB.MockSourceIPAddress = "2.2.2.2"

	testFlowA.MockSourcePort = 30000
	testFlowB.MockSourcePort = 4444

	testFlowA.MockDestinationIPAddress = "2.2.2.2"
	testFlowB.MockDestinationIPAddress = "1.1.1.1"

	testFlowA.MockDestinationPort = 4444
	testFlowB.MockDestinationPort = 30000

	testFlowA.MockProtocolIdentifier = protocols.UDP
	testFlowB.MockProtocolIdentifier = protocols.UDP

	testFlowB.MockExporter = testFlowA.Exporter()

	testFlowA.MockFlowStartMilliseconds = 100
	testFlowA.MockFlowEndMilliseconds = 200

	testFlowB.MockFlowStartMilliseconds = 300
	testFlowB.MockFlowEndMilliseconds = 400

	var sessA session.Aggregate
	var sessB session.Aggregate
	session.FromFlow(testFlowA, &sessA)
	session.FromFlow(testFlowB, &sessB)
	err := sessA.Merge(&sessB)

	require.Nil(t, err)

	//Don't mess with the flow key
	require.Equal(t, testFlowA.SourceIPAddress(), sessA.IPAddressA)
	require.Equal(t, testFlowA.SourcePort(), sessA.PortA)
	require.Equal(t, testFlowA.DestinationIPAddress(), sessA.IPAddressB)
	require.Equal(t, testFlowA.DestinationPort(), sessA.PortB)
	require.Equal(t, testFlowA.ProtocolIdentifier(), sessA.ProtocolIdentifier)
	require.Equal(t, testFlowA.Exporter(), sessA.Exporter)

	require.Equal(t, testFlowA.MockFlowStartMilliseconds, sessA.FlowStartMillisecondsAB)
	require.Equal(t, testFlowA.MockFlowEndMilliseconds, sessA.FlowEndMillisecondsAB)

	require.Equal(t, testFlowB.MockFlowStartMilliseconds, sessA.FlowStartMillisecondsBA)
	require.Equal(t, testFlowB.MockFlowEndMilliseconds, sessA.FlowEndMillisecondsBA)

	require.Equal(t, testFlowA.OctetTotalCount(), sessA.OctetTotalCountAB)
	require.Equal(t, testFlowA.PacketTotalCount(), sessA.PacketTotalCountAB)

	require.Equal(t, testFlowB.OctetTotalCount(), sessA.OctetTotalCountBA)
	require.Equal(t, testFlowB.PacketTotalCount(), sessA.PacketTotalCountBA)

	require.Equal(t, testFlowA.FlowEndReason(), sessA.FlowEndReasonAB)

	require.Equal(t, testFlowB.FlowEndReason(), sessB.FlowEndReasonBA)
}

func TestToRitaConnABSrcDest(t *testing.T) {
	testFlowA := ipfix.NewFlowMock()
	testFlowB := ipfix.NewFlowMock()

	testFlowA.MockSourceIPAddress = "1.1.1.1"
	testFlowB.MockSourceIPAddress = "2.2.2.2"

	testFlowA.MockSourcePort = 30000
	testFlowB.MockSourcePort = 4444

	testFlowA.MockDestinationIPAddress = "2.2.2.2"
	testFlowB.MockDestinationIPAddress = "1.1.1.1"

	testFlowA.MockDestinationPort = 4444
	testFlowB.MockDestinationPort = 30000

	testFlowA.MockProtocolIdentifier = protocols.UDP
	testFlowB.MockProtocolIdentifier = protocols.UDP

	testFlowB.MockExporter = testFlowA.Exporter()

	testFlowA.MockFlowStartMilliseconds = 100
	testFlowA.MockFlowEndMilliseconds = 200

	testFlowB.MockFlowStartMilliseconds = 300
	testFlowB.MockFlowEndMilliseconds = 400

	var sessA session.Aggregate
	var sessB session.Aggregate
	session.FromFlow(testFlowA, &sessA)
	session.FromFlow(testFlowB, &sessB)
	err := sessA.Merge(&sessB)

	require.Nil(t, err)

	var conn parsetypes.Conn
	sessA.ToRITAConn(&conn, func(arg1 string) bool {
		return arg1 == "1.1.1.1"
	})

	require.Equal(t, testFlowA.SourceIPAddress(), conn.Source)
	require.Equal(t, testFlowA.SourcePort(), uint16(conn.SourcePort))
	require.Equal(t, testFlowA.DestinationIPAddress(), conn.Destination)
	require.Equal(t, testFlowA.DestinationPort(), uint16(conn.DestinationPort))
	require.Equal(t, "udp", conn.Proto)

	require.Equal(t, testFlowA.MockFlowStartMilliseconds/1000, uint64(conn.TimeStamp))
	require.NotZero(t, conn.Duration)
	require.Equal(
		t,
		float64(testFlowB.MockFlowEndMilliseconds-testFlowA.MockFlowStartMilliseconds)/1000.0,
		conn.Duration,
	)

	require.Equal(t, testFlowA.OctetTotalCount(), uint64(conn.OrigIPBytes))
	require.Equal(t, testFlowA.PacketTotalCount(), uint64(conn.OrigPkts))

	require.Equal(t, testFlowB.OctetTotalCount(), uint64(conn.RespIPBytes))
	require.Equal(t, testFlowB.PacketTotalCount(), uint64(conn.RespPkts))

	require.True(t, conn.LocalOrigin)
	require.False(t, conn.LocalResponse)

	require.Zero(t, conn.ConnState)
	require.Zero(t, conn.History)
	require.Zero(t, conn.MissedBytes)
	require.Zero(t, conn.Service)
	require.Zero(t, conn.OrigBytes)
	require.Zero(t, conn.RespBytes)
	require.Zero(t, conn.UID)
	require.Len(t, conn.TunnelParents, 0)
}

func TestToRitaConnBASrcDest(t *testing.T) {
	testFlowA := ipfix.NewFlowMock()
	testFlowB := ipfix.NewFlowMock()

	testFlowA.MockSourceIPAddress = "1.1.1.1"
	testFlowB.MockSourceIPAddress = "2.2.2.2"

	testFlowA.MockSourcePort = 30000
	testFlowB.MockSourcePort = 4444

	testFlowA.MockDestinationIPAddress = "2.2.2.2"
	testFlowB.MockDestinationIPAddress = "1.1.1.1"

	testFlowA.MockDestinationPort = 4444
	testFlowB.MockDestinationPort = 30000

	testFlowA.MockProtocolIdentifier = protocols.UDP
	testFlowB.MockProtocolIdentifier = protocols.UDP

	testFlowB.MockExporter = testFlowA.Exporter()

	testFlowA.MockFlowStartMilliseconds = 300
	testFlowA.MockFlowEndMilliseconds = 400

	testFlowB.MockFlowStartMilliseconds = 100
	testFlowB.MockFlowEndMilliseconds = 200

	var sessA session.Aggregate
	var sessB session.Aggregate
	session.FromFlow(testFlowA, &sessA)
	session.FromFlow(testFlowB, &sessB)
	err := sessA.Merge(&sessB)

	require.Nil(t, err)

	var conn parsetypes.Conn
	sessA.ToRITAConn(&conn, func(arg1 string) bool {
		return arg1 == "2.2.2.2"
	})

	require.Equal(t, testFlowB.SourceIPAddress(), conn.Source)
	require.Equal(t, testFlowB.SourcePort(), uint16(conn.SourcePort))
	require.Equal(t, testFlowB.DestinationIPAddress(), conn.Destination)
	require.Equal(t, testFlowB.DestinationPort(), uint16(conn.DestinationPort))
	require.Equal(t, "udp", conn.Proto)

	require.Equal(t, testFlowB.MockFlowStartMilliseconds/1000, uint64(conn.TimeStamp))
	require.NotZero(t, conn.Duration)
	require.Equal(
		t,
		float64(testFlowA.MockFlowEndMilliseconds-testFlowB.MockFlowStartMilliseconds)/1000.0,
		conn.Duration,
	)

	require.Equal(t, testFlowB.OctetTotalCount(), uint64(conn.OrigIPBytes))
	require.Equal(t, testFlowB.PacketTotalCount(), uint64(conn.OrigPkts))

	require.Equal(t, testFlowA.OctetTotalCount(), uint64(conn.RespIPBytes))
	require.Equal(t, testFlowA.PacketTotalCount(), uint64(conn.RespPkts))

	require.True(t, conn.LocalOrigin)
	require.False(t, conn.LocalResponse)

	require.Zero(t, conn.ConnState)
	require.Zero(t, conn.History)
	require.Zero(t, conn.MissedBytes)
	require.Zero(t, conn.Service)
	require.Zero(t, conn.OrigBytes)
	require.Zero(t, conn.RespBytes)
	require.Zero(t, conn.UID)
	require.Len(t, conn.TunnelParents, 0)
}

func TestToRITAProtos(t *testing.T) {
	var conn parsetypes.Conn

	sess := session.Aggregate{
		ProtocolIdentifier: protocols.TCP,
	}
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "tcp", conn.Proto)

	sess = session.Aggregate{
		ProtocolIdentifier: protocols.UDP,
	}
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "udp", conn.Proto)

	sess = session.Aggregate{
		ProtocolIdentifier: protocols.ICMP,
	}
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "icmp", conn.Proto)

	sess = session.Aggregate{
		ProtocolIdentifier: protocols.IPv6_ICMP,
	}
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "icmp", conn.Proto)

	sess = session.Aggregate{
		ProtocolIdentifier: protocols.MPLS_IN_IP,
	}
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "unknown_transport", conn.Proto)
}
