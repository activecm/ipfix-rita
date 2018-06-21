package session

import (
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

//Aggregate is used to aggregate multiple flows in to a session.
//Note the fields are marked A, B, AB, and BA. These markers take the
//place of source and destination. The source of a session is found
//by comparing the FlowStart times from host A to B and from host B to A.
//The originating host of whichever flow is earlier is the source.
//
//In order to remove ambiguity, IPAddressA must come before IPAddressB
//alphabetically. Otherwise the same session could be represented by
//two different session aggregates.
type Aggregate struct {
	AggregateQuery `bson:",inline"`
	ID             bson.ObjectId `bson:"_id,omitempty"`

	FlowStartMillisecondsAB uint64 `bson:"flowStartMillisecondsAB"`
	FlowEndMillisecondsAB   uint64 `bson:"flowEndMillisecondsAB"`
	FlowStartMillisecondsBA uint64 `bson:"flowStartMillisecondsBA"`
	FlowEndMillisecondsBA   uint64 `bson:"flowEndMillisecondsBA"`

	OctetTotalCountAB  uint64 `bson:"octetTotalCountAB"`
	OctetTotalCountBA  uint64 `bson:"octetTotalCountBA"`
	PacketTotalCountAB uint64 `bson:"packetTotalCountAB"`
	PacketTotalCountBA uint64 `bson:"packetTotalCountBA"`

	FlowEndReasonAB ipfix.FlowEndReason `bson:"flowEndReasonAB"`
	FlowEndReasonBA ipfix.FlowEndReason `bson:"flowEndReasonBA"`
}

//AggregateQuery represents the Flow Key + Exporter used to uniquely
//identify each session aggregate
type AggregateQuery struct {
	IPAddressA string `bson:"IPAddressA"`
	PortA      uint16 `bson:"transportPortA"`

	IPAddressB string `bson:"IPAddressB"`
	PortB      uint16 `bson:"transportPortB"`

	ProtocolIdentifier protocols.Identifier `bson:"protocolIdentifier"`

	Exporter string `bson:"exporter"`
}

//FlowAggregateAssignment describes whether a Flow's source and destination
//are assigned to host A and B respectively, or if they are assigned
//to host B and A respectively
type FlowAggregateAssignment bool

//ASource signifies the flow's source was mapped to IPAddressA
const ASource FlowAggregateAssignment = true

//BSource signifies the flow's source was mapped to IPAddressB
const BSource FlowAggregateAssignment = false

//FromFlow fills a SessionAggregate from a Flow.
//Note: ID is unaffected by this function.
func FromFlow(flow ipfix.Flow, sess *Aggregate) (FlowAggregateAssignment, error) {
	flowSource := flow.SourceIPAddress()
	flowDest := flow.DestinationIPAddress()

	flowStart, err := flow.FlowStartMilliseconds()
	if err != nil {
		return ASource, err
	}

	flowEnd, err := flow.FlowEndMilliseconds()
	if err != nil {
		return ASource, err
	}

	sess.ProtocolIdentifier = flow.ProtocolIdentifier()
	sess.Exporter = flow.Exporter()

	if flowSource < flowDest {
		//flowSource is IPAddressA
		sess.IPAddressA = flowSource
		sess.PortA = flow.SourcePort()
		sess.IPAddressB = flowDest
		sess.PortB = flow.DestinationPort()
		sess.FlowStartMillisecondsAB = flowStart
		sess.FlowEndMillisecondsAB = flowEnd
		sess.OctetTotalCountAB = flow.OctetTotalCount()
		sess.PacketTotalCountAB = flow.PacketTotalCount()
		sess.FlowEndReasonAB = flow.FlowEndReason()
		sess.FlowEndReasonBA = ipfix.Nil
		return ASource, nil
	}
	//flowDest is IPAddressA
	sess.IPAddressA = flowDest
	sess.PortA = flow.DestinationPort()
	sess.IPAddressB = flowSource
	sess.PortB = flow.SourcePort()
	sess.FlowStartMillisecondsBA = flowStart
	sess.FlowEndMillisecondsBA = flowEnd
	sess.OctetTotalCountBA = flow.OctetTotalCount()
	sess.PacketTotalCountBA = flow.PacketTotalCount()
	sess.FlowEndReasonBA = flow.FlowEndReason()
	sess.FlowEndReasonAB = ipfix.Nil
	return BSource, nil
}

//Merge merges another aggregate into this aggregate
func (s *Aggregate) Merge(other *Aggregate) error {
	if s.IPAddressA != other.IPAddressA ||
		s.IPAddressB != other.IPAddressB ||
		s.PortA != other.PortA ||
		s.PortB != other.PortB ||
		s.ProtocolIdentifier != other.ProtocolIdentifier ||
		s.Exporter != other.Exporter {
		return errors.New("cannot merge flows with different flow keys")
	}

	s.OctetTotalCountAB += other.OctetTotalCountAB
	s.OctetTotalCountBA += other.OctetTotalCountBA
	s.PacketTotalCountAB += other.PacketTotalCountAB
	s.PacketTotalCountBA += other.PacketTotalCountBA

	//if other has the field set, and s doesn't or other's is earlier
	if other.FlowStartMillisecondsAB != 0 && (s.FlowStartMillisecondsAB == 0 ||
		other.FlowStartMillisecondsAB < s.FlowStartMillisecondsAB) {
		s.FlowStartMillisecondsAB = other.FlowStartMillisecondsAB
	}

	if other.FlowStartMillisecondsBA != 0 && (s.FlowStartMillisecondsBA == 0 ||
		other.FlowStartMillisecondsBA < s.FlowStartMillisecondsBA) {
		s.FlowStartMillisecondsBA = other.FlowStartMillisecondsBA
	}

	//if other has the field set, and s other's is later
	//we don't have to check if s's field is unset since the is later condition
	//covers it
	if other.FlowEndMillisecondsAB != 0 &&
		other.FlowEndMillisecondsAB > s.FlowEndMillisecondsAB {
		s.FlowEndMillisecondsAB = other.FlowEndMillisecondsAB
		s.FlowEndReasonAB = other.FlowEndReasonAB
	}

	if other.FlowEndMillisecondsBA != 0 &&
		other.FlowEndMillisecondsBA > s.FlowEndMillisecondsBA {
		s.FlowEndMillisecondsBA = other.FlowEndMillisecondsBA
		s.FlowEndReasonBA = other.FlowEndReasonBA
	}

	return nil
}

//Clear sets an aggregate to its empty state
func (s *Aggregate) Clear() {
	s.ID = bson.ObjectId("")

	s.IPAddressA = ""
	s.PortA = 0

	s.IPAddressB = ""
	s.PortB = 0

	s.ProtocolIdentifier = protocols.Identifier(0)

	s.Exporter = ""

	s.FlowStartMillisecondsAB = 0
	s.FlowEndMillisecondsAB = 0
	s.FlowStartMillisecondsBA = 0
	s.FlowEndMillisecondsBA = 0

	s.OctetTotalCountAB = 0
	s.OctetTotalCountBA = 0
	s.PacketTotalCountAB = 0
	s.PacketTotalCountBA = 0

	s.FlowEndReasonAB = ipfix.Nil
	s.FlowEndReasonBA = ipfix.Nil
}

//ToRITAConn fills a RITA Conn record with the data held by the session aggregate.
//localFunc is used to decide whether to mark an IP address as local or not.
func (s *Aggregate) ToRITAConn(conn *parsetypes.Conn, localFunc func(string) bool) {
	conn.UID = ""
	conn.Service = ""
	conn.ConnState = ""
	conn.OrigBytes = 0 // Not used (OrigIPBytes is used instead)
	conn.RespBytes = 0 // Not used (RespIPBytes is used instead)
	conn.MissedBytes = 0
	conn.History = ""
	conn.TunnelParents = []string{}

	switch s.ProtocolIdentifier {
	case protocols.TCP:
		conn.Proto = "tcp"
	case protocols.UDP:
		conn.Proto = "udp"
	case protocols.ICMP:
		fallthrough
	case protocols.IPv6_ICMP:
		conn.Proto = "icmp"
	default:
		conn.Proto = "unknown_transport"
	}

	sessionEnd := s.FlowEndMillisecondsAB
	if s.FlowEndMillisecondsBA > sessionEnd {
		sessionEnd = s.FlowEndMillisecondsBA
	}

	//if a started sending data before b, then a is the source
	if s.FlowStartMillisecondsAB < s.FlowStartMillisecondsBA {
		//host a is source
		sessionStart := s.FlowStartMillisecondsAB
		conn.TimeStamp = int64(sessionStart / 1000)
		conn.Duration = float64(sessionEnd-sessionStart) / 1000.0

		conn.Source = s.IPAddressA
		conn.SourcePort = int(s.PortA)
		conn.Destination = s.IPAddressB
		conn.DestinationPort = int(s.PortB)
		conn.LocalOrigin = localFunc(s.IPAddressA)
		conn.LocalResponse = localFunc(s.IPAddressB)
		conn.OrigPkts = int64(s.PacketTotalCountAB)
		conn.RespPkts = int64(s.PacketTotalCountBA)
		conn.OrigIPBytes = int64(s.OctetTotalCountAB)
		conn.RespIPBytes = int64(s.OctetTotalCountBA)
	} else {
		//host b is source
		sessionStart := s.FlowStartMillisecondsBA
		conn.TimeStamp = int64(sessionStart / 1000)
		conn.Duration = float64(sessionEnd-sessionStart) / 1000.0

		conn.Source = s.IPAddressB
		conn.SourcePort = int(s.PortB)
		conn.Destination = s.IPAddressA
		conn.DestinationPort = int(s.PortA)
		conn.LocalOrigin = localFunc(s.IPAddressB)
		conn.LocalResponse = localFunc(s.IPAddressA)
		conn.OrigPkts = int64(s.PacketTotalCountBA)
		conn.RespPkts = int64(s.PacketTotalCountAB)
		conn.OrigIPBytes = int64(s.OctetTotalCountBA)
		conn.RespIPBytes = int64(s.OctetTotalCountAB)
	}
}
