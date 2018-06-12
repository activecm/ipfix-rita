package ipfix

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/types/protocols"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

//MongoDBLogstash represents an MongoDBLogstash record stored in MongoDB via MongoDBLogstash
type MongoDBLogstash struct {
	ID      bson.ObjectId `bson:"_id"`
	Host    string        `bson:"host"` //Host is the metering process host
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

		OctetTotalCount  uint64 `bson:"octetTotalCount"`
		PacketTotalCount uint64 `bson:"packetTotalCount"`

		ProtocolIdentifier protocols.Identifier `bson:"protocolIdentifier"`
		IPClassOfService   uint8                `bson:"ipClassOfService"`
		VlanID             uint16               `bson:"vlanID"`
		FlowEndReason      FlowEndReason        `bson:"flowEndReason"`
		Version            uint8                `bson:"version"`
	} `bson:"netflow"`
}

//SourceIPAddress returns the source IPv4 or IPv6 address
func (i *MongoDBLogstash) SourceIPAddress() string {
	if len(i.Netflow.SourceIPv4) != 0 {
		return i.Netflow.SourceIPv4
	}
	return i.Netflow.SourceIPv6
}

//SourcePort returns the source transport port
func (i *MongoDBLogstash) SourcePort() uint16 {
	return i.Netflow.SourcePort
}

//DestinationIPAddress returns the destination IPv4 or IPv6 address
func (i *MongoDBLogstash) DestinationIPAddress() string {
	if len(i.Netflow.DestinationIPv4) != 0 {
		return i.Netflow.DestinationIPv4
	}
	return i.Netflow.DestinationIPv6
}

//DestinationPort returns the destination transport port
func (i *MongoDBLogstash) DestinationPort() uint16 {
	return i.Netflow.DestinationPort
}

//ProtocolIdentifier returns which transport protocol was used
func (i *MongoDBLogstash) ProtocolIdentifier() protocols.Identifier {
	return i.Netflow.ProtocolIdentifier
}

//IPClassOfService is the value of the TOS field in the IPv4 packet header or
//the value of the Traffic Class field in the IPv6 packet header.
func (i *MongoDBLogstash) IPClassOfService() uint8 {
	return i.Netflow.IPClassOfService
}

//FlowStartMilliseconds is the time the flow started as a Unix timestamp
func (i *MongoDBLogstash) FlowStartMilliseconds() (uint64, error) {
	t, err := time.Parse(time.RFC3339Nano, i.Netflow.FlowStartMilliseconds)
	err = errors.WithStack(err)
	if err != nil {
		return 0, err
	}
	return uint64(t.UnixNano() / 1000000), nil
}

//FlowEndMilliseconds is the time the flow ended as a Unix timestamp
func (i *MongoDBLogstash) FlowEndMilliseconds() (uint64, error) {
	t, err := time.Parse(time.RFC3339Nano, i.Netflow.FlowEndMilliseconds)
	err = errors.WithStack(err)
	if err != nil {
		return 0, err
	}
	return uint64(t.UnixNano() / 1000000), nil
}

//OctetTotalCount returns the total amount of bytes sent (including IP headers and payload)
func (i *MongoDBLogstash) OctetTotalCount() uint64 {
	return i.Netflow.OctetTotalCount
}

//PacketTotalCount returns the number of packets sent from the source to the destination
func (i *MongoDBLogstash) PacketTotalCount() uint64 {
	return i.Netflow.PacketTotalCount
}

//VlanID returns which Vlan the flow took place on at the time of observation
func (i *MongoDBLogstash) VlanID() uint16 {
	return i.Netflow.VlanID
}

//FlowEndReason returns why the metering process stopped recording the flow
func (i *MongoDBLogstash) FlowEndReason() FlowEndReason {
	return i.Netflow.FlowEndReason
}

//Version returns the IPFIX/Netflow version
func (i *MongoDBLogstash) Version() uint8 {
	return i.Netflow.Version
}

//Exporter returns the address of the exporting process for this flow
func (i *MongoDBLogstash) Exporter() string {
	return i.Host
}
