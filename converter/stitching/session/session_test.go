package session_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/stretchr/testify/require"
	"github.com/globalsign/mgo/bson"
)

func TestFromFlowASource(t *testing.T) {
	var sess session.Aggregate
	testFlow := input.NewFlowMock()
	testFlow.MockSourceIPAddress = "1.1.1.1"
	testFlow.MockDestinationIPAddress = "2.2.2.2"
	err := session.FromFlow(testFlow, &sess)
	require.Nil(t, err)
	require.True(t, sess.FilledFromSourceA)
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

func TestFromFlowBSource(t *testing.T) {
	var sess session.Aggregate
	testFlow := input.NewFlowMock()
	testFlow.MockSourceIPAddress = "2.2.2.2"
	testFlow.MockDestinationIPAddress = "1.1.1.1"
	err := session.FromFlow(testFlow, &sess)
	require.Nil(t, err)
	require.True(t, sess.FilledFromSourceB)
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
	testFlow := input.NewFlowMock()
	session.FromFlow(testFlow, &sess)

	//ensure there is data
	require.Equal(t, testFlow.Exporter(), sess.Exporter)

	sess.Clear()
	require.Equal(t, nil, sess.MatcherID)
	require.False(t, sess.FilledFromSourceA)
	require.False(t, sess.FilledFromSourceB)
	require.Equal(t, "", sess.IPAddressA)
	require.Equal(t, "", sess.IPAddressB)
	require.Equal(t, uint16(0), sess.PortA)
	require.Equal(t, uint16(0), sess.PortB)
	require.Equal(t, protocols.Identifier(0), sess.ProtocolIdentifier)
	require.Equal(t, "", sess.Exporter)
	require.Equal(t, int64(0), sess.FlowStartMillisecondsAB)
	require.Equal(t, int64(0), sess.FlowStartMillisecondsBA)
	require.Equal(t, int64(0), sess.FlowEndMillisecondsAB)
	require.Equal(t, int64(0), sess.FlowEndMillisecondsBA)
	require.Equal(t, int64(0), sess.OctetTotalCountAB)
	require.Equal(t, int64(0), sess.OctetTotalCountBA)
	require.Equal(t, int64(0), sess.PacketTotalCountAB)
	require.Equal(t, int64(0), sess.PacketTotalCountBA)
	require.Equal(t, input.NilEndReason, sess.FlowEndReasonAB)
	require.Equal(t, input.NilEndReason, sess.FlowEndReasonBA)
}

func TestMergeWrongFlowKeys(t *testing.T) {
	testFlowA := input.NewFlowMock()
	testFlowB := input.NewFlowMock()

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
	testFlowA := input.NewFlowMock()
	testFlowB := input.NewFlowMock()

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
	sessA.MatcherID = bson.NewObjectId()
	sessAMatcherIDCopy := sessA.MatcherID
	sessB.MatcherID = bson.NewObjectId()
	err := sessA.Merge(&sessB)

	require.Nil(t, err)

	require.True(t, sessA.FilledFromSourceA)
	require.False(t, sessA.FilledFromSourceB)

	//Dont' mess with the object MatcherID
	require.Equal(t, sessAMatcherIDCopy, sessA.MatcherID)

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
	testFlowA := input.NewFlowMock()
	testFlowB := input.NewFlowMock()

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
	sessA.MatcherID = bson.NewObjectId()
	sessAMatcherIDCopy := sessA.MatcherID
	sessB.MatcherID = bson.NewObjectId()
	err := sessA.Merge(&sessB)

	require.Nil(t, err)
	require.Equal(t, sessAMatcherIDCopy, sessA.MatcherID)
	require.True(t, sessA.FilledFromSourceA)
	require.False(t, sessA.FilledFromSourceB)
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
	testFlowA := input.NewFlowMock()
	testFlowB := input.NewFlowMock()

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
	sessA.MatcherID = bson.NewObjectId()
	sessAMatcherIDCopy := sessA.MatcherID
	sessB.MatcherID = bson.NewObjectId()
	err := sessA.Merge(&sessB)

	require.Nil(t, err)
	require.Equal(t, sessAMatcherIDCopy, sessA.MatcherID)
	require.True(t, sessA.FilledFromSourceA)
	require.True(t, sessA.FilledFromSourceB)
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

//TODO: TestToRITASingleFlow

func TestToRitaConnABSrcDest(t *testing.T) {
	testFlowA := input.NewFlowMock()
	testFlowB := input.NewFlowMock()

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

	require.Equal(t, testFlowA.MockFlowStartMilliseconds/1000, int64(conn.TimeStamp))
	require.NotZero(t, conn.Duration)
	require.Equal(
		t,
		float64(testFlowB.MockFlowEndMilliseconds-testFlowA.MockFlowStartMilliseconds)/1000.0,
		conn.Duration,
	)

	require.Equal(t, testFlowA.OctetTotalCount(), int64(conn.OrigIPBytes))
	require.Equal(t, testFlowA.PacketTotalCount(), int64(conn.OrigPkts))

	require.Equal(t, testFlowB.OctetTotalCount(), int64(conn.RespIPBytes))
	require.Equal(t, testFlowB.PacketTotalCount(), int64(conn.RespPkts))

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
	testFlowA := input.NewFlowMock()
	testFlowB := input.NewFlowMock()

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

	require.Equal(t, testFlowB.MockFlowStartMilliseconds/1000, int64(conn.TimeStamp))
	require.NotZero(t, conn.Duration)
	require.Equal(
		t,
		float64(testFlowA.MockFlowEndMilliseconds-testFlowB.MockFlowStartMilliseconds)/1000.0,
		conn.Duration,
	)

	require.Equal(t, testFlowB.OctetTotalCount(), int64(conn.OrigIPBytes))
	require.Equal(t, testFlowB.PacketTotalCount(), int64(conn.OrigPkts))

	require.Equal(t, testFlowA.OctetTotalCount(), int64(conn.RespIPBytes))
	require.Equal(t, testFlowA.PacketTotalCount(), int64(conn.RespPkts))

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

	sess := session.Aggregate{}
	sess.ProtocolIdentifier = protocols.TCP
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "tcp", conn.Proto)

	sess = session.Aggregate{}
	sess.ProtocolIdentifier = protocols.UDP
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "udp", conn.Proto)

	sess = session.Aggregate{}
	sess.ProtocolIdentifier = protocols.ICMP
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "icmp", conn.Proto)

	sess = session.Aggregate{}
	sess.ProtocolIdentifier = protocols.IPv6_ICMP
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "icmp", conn.Proto)

	sess = session.Aggregate{}
	sess.ProtocolIdentifier = protocols.MPLS_IN_IP
	sess.ToRITAConn(&conn, func(arg1 string) bool { return false })
	require.Equal(t, "unknown_transport", conn.Proto)
}

/*
//not needed since MongoMatch (a Matcher based on MongoDB) was removed

func TestMongoDBStorage(t *testing.T) {
	//clear out the Sessions collection used by the MongoMatcher
	integrationtest.RegisterDependenciesResetFunc(
		func(t *testing.T, deps *integrationtest.Dependencies) {
			sessionsColl := deps.Env.DB.NewHelperCollection(mongomatch.SessionsCollName)
			_, err := sessionsColl.RemoveAll(nil)
			if err != nil {
				deps.Env.Error(err, nil)
				t.FailNow()
			}
			sessionsColl.Database.Session.Close()
		},
	)
	env := integrationtest.GetDependencies(t).Env
	defer integrationtest.CloseDependencies()

	testFlowA := input.NewFlowMock()
	testFlowB := input.NewFlowMock()

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

	//store sessA as it stands
	sessionsColl := env.DB.NewHelperCollection(mongomatch.SessionsCollName)
	err := sessionsColl.Insert(&sessA)
	require.Nil(t, err)

	//ensure sessA was stored correctly and is able to be retrieved
	var storedSessA session.Aggregate
	err = sessionsColl.Find(&sessA.AggregateQuery).One(&storedSessA)
	require.Nil(t, err)

	//The id was set by mongodb
	sessA.MatcherID = storedSessA.MatcherID

	//make sure sessA comes back the same (create, find, read)
	require.Equal(t, sessA, storedSessA)

	//merge in sessB and update the database
	err = sessA.Merge(&sessB)
	require.Nil(t, err)

	var storedSessPreUpdate session.Aggregate
	info, err := sessionsColl.Find(&sessA.AggregateQuery).Apply(mgo.Change{
		Upsert: true,
		Update: &sessA,
	}, &storedSessPreUpdate)

	require.Nil(t, err)
	require.Equal(t, 1, info.Matched)
	require.Equal(t, storedSessA, storedSessPreUpdate)

	//verify update actually happened
	var storedSessPostUpdate session.Aggregate
	err = sessionsColl.Find(&sessA.AggregateQuery).One(&storedSessPostUpdate)
	require.Nil(t, err)
	require.Equal(t, sessA, storedSessPostUpdate)
}
*/
