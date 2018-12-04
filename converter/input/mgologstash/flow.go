package mgologstash

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

//Flow represents an IPFIX/ Netflowv9 flow record stored in MongoDB via Logstash.
//The bson tags are given for an IPFIX flow.
type Flow struct {
	ID      bson.ObjectId `bson:"_id,omitempty"` //12 bytes
	Host    string        `bson:"host"`          //Host is the metering process host (24 bytes)
	Netflow struct {
		SourceIPv4 string `bson:"sourceIPv4Address,omitempty"`
		SourceIPv6 string `bson:"sourceIPv6Address,omitempty"`
		SourcePort uint16 `bson:"sourceTransportPort"`

		DestinationIPv4 string `bson:"destinationIPv4Address,omitempty"`
		DestinationIPv6 string `bson:"destinationIPv6Address,omitempty"`
		DestinationPort uint16 `bson:"destinationTransportPort"`

		// NOTE: We may need fields for other time units
		FlowStartMilliseconds string `bson:"flowStartMilliseconds"`
		FlowEndMilliseconds   string `bson:"flowEndMilliseconds"`

		OctetTotalCount  int64 `bson:"octetTotalCount"`
		PacketTotalCount int64 `bson:"packetTotalCount"`

		ProtocolIdentifier protocols.Identifier `bson:"protocolIdentifier"`
		FlowEndReason      input.FlowEndReason  `bson:"flowEndReason"`
		Version            uint8                `bson:"version"`
	} `bson:"netflow"`
}

//SourceIPAddress returns the source IPv4 or IPv6 address
func (i *Flow) SourceIPAddress() string {
	if len(i.Netflow.SourceIPv4) != 0 {
		return i.Netflow.SourceIPv4
	}
	return i.Netflow.SourceIPv6
}

//SourcePort returns the source transport port
func (i *Flow) SourcePort() uint16 {
	return i.Netflow.SourcePort
}

//DestinationIPAddress returns the destination IPv4 or IPv6 address
func (i *Flow) DestinationIPAddress() string {
	if len(i.Netflow.DestinationIPv4) != 0 {
		return i.Netflow.DestinationIPv4
	}
	return i.Netflow.DestinationIPv6
}

//DestinationPort returns the destination transport port
func (i *Flow) DestinationPort() uint16 {
	return i.Netflow.DestinationPort
}

//ProtocolIdentifier returns which transport protocol was used
func (i *Flow) ProtocolIdentifier() protocols.Identifier {
	return i.Netflow.ProtocolIdentifier
}

//FlowStartMilliseconds is the time the flow started as a Unix timestamp
func (i *Flow) FlowStartMilliseconds() (int64, error) {
	t, err := time.Parse(time.RFC3339Nano, i.Netflow.FlowStartMilliseconds)
	err = errors.WithStack(err)
	if err != nil {
		return 0, err
	}
	return int64(t.UnixNano() / 1000000), nil
}

//FlowEndMilliseconds is the time the flow ended as a Unix timestamp
func (i *Flow) FlowEndMilliseconds() (int64, error) {
	t, err := time.Parse(time.RFC3339Nano, i.Netflow.FlowEndMilliseconds)
	err = errors.WithStack(err)
	if err != nil {
		return 0, err
	}
	return int64(t.UnixNano() / 1000000), nil
}

//OctetTotalCount returns the total amount of bytes sent (including IP headers and payload)
func (i *Flow) OctetTotalCount() int64 {
	return i.Netflow.OctetTotalCount
}

//PacketTotalCount returns the number of packets sent from the source to the destination
func (i *Flow) PacketTotalCount() int64 {
	return i.Netflow.PacketTotalCount
}

//FlowEndReason returns why the metering process stopped recording the flow
func (i *Flow) FlowEndReason() input.FlowEndReason {
	return i.Netflow.FlowEndReason
}

//Version returns the IPFIX/Netflow version
func (i *Flow) Version() uint8 {
	return i.Netflow.Version
}

//Exporter returns the address of the exporting process for this flow
func (i *Flow) Exporter() string {
	return i.Host
}
