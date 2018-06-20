package ipfix

import (
	"math"
	"math/rand"
	"strconv"

	"github.com/activecm/ipfix-rita/converter/protocols"
)

//FlowMock is a convenient struct for represeting Flow data
type FlowMock struct {
	MockExporter        string
	MockSourceIPAddress string
	MockSourcePort      uint16

	MockDestinationIPAddress string
	MockDestinationPort      uint16

	MockFlowStartMilliseconds uint64
	MockFlowEndMilliseconds   uint64

	MockOctetTotalCount  uint64
	MockPacketTotalCount uint64

	MockProtocolIdentifier protocols.Identifier
	MockIPClassOfService   uint8
	MockVlanID             uint16
	MockFlowEndReason      FlowEndReason
	MockVersion            uint8
}

//NewFlowMock returns a ipfix.Flow with random data
func NewFlowMock() *FlowMock {
	randIP := func() string {
		b := make([]byte, 0)
		strconv.AppendInt(b, rand.Int63n(256), 10)
		b = append(b, '.')
		strconv.AppendInt(b, rand.Int63n(256), 10)
		b = append(b, '.')
		strconv.AppendInt(b, rand.Int63n(256), 10)
		b = append(b, '.')
		strconv.AppendInt(b, rand.Int63n(256), 10)
		return string(b)
	}
	randShort := func() uint16 {
		a := rand.Int31() % math.MaxUint16
		if a < 0 {
			a = -a
		}
		return uint16(a)
	}
	randByte := func() uint8 {
		a := rand.Int31() % math.MaxUint8
		if a < 0 {
			a = -a
		}
		return uint8(a)
	}
	return &FlowMock{
		MockExporter:              randIP(),
		MockSourceIPAddress:       randIP(),
		MockSourcePort:            randShort(),
		MockDestinationIPAddress:  randIP(),
		MockDestinationPort:       randShort(),
		MockFlowStartMilliseconds: uint64(rand.Uint32()),
		MockFlowEndMilliseconds:   uint64(rand.Uint32()),
		MockOctetTotalCount:       uint64(rand.Uint32()),
		MockPacketTotalCount:      uint64(rand.Uint32()),
		MockProtocolIdentifier:    protocols.Identifier(randByte()),
		MockIPClassOfService:      randByte(),
		MockVlanID:                randShort(),
		MockFlowEndReason:         FlowEndReason(rand.Intn(4)),
		MockVersion:               10,
	}
}

//SourceIPAddress returns the source IPv4 or IPv6 address
func (f *FlowMock) SourceIPAddress() string {
	return f.MockSourceIPAddress
}

//SourcePort returns the source transport port
func (f *FlowMock) SourcePort() uint16 {
	return f.MockSourcePort
}

//DestinationIPAddress returns the destination IPv4 or IPv6 address
func (f *FlowMock) DestinationIPAddress() string {
	return f.MockDestinationIPAddress
}

//DestinationPort returns the destination transport port
func (f *FlowMock) DestinationPort() uint16 {
	return f.MockDestinationPort
}

//ProtocolIdentifier returns which transport protocol was used
func (f *FlowMock) ProtocolIdentifier() protocols.Identifier {
	return f.MockProtocolIdentifier
}

//IPClassOfService is the value of the TOS field in the IPv4 packet header or
//the value of the Traffic Class field in the IPv6 packet header.
func (f *FlowMock) IPClassOfService() uint8 {
	return f.MockIPClassOfService
}

//FlowStartMilliseconds is the time the flow started as a Unix timestamp
func (f *FlowMock) FlowStartMilliseconds() (uint64, error) {
	return f.MockFlowStartMilliseconds, nil
}

//FlowEndMilliseconds is the time the flow ended as a Unix timestamp
func (f *FlowMock) FlowEndMilliseconds() (uint64, error) {
	return f.MockFlowEndMilliseconds, nil
}

//OctetTotalCount returns the total amount of bytes sent (including IP headers and payload)
func (f *FlowMock) OctetTotalCount() uint64 {
	return f.MockOctetTotalCount
}

//PacketTotalCount returns the number of packets sent from the source to the destination
func (f *FlowMock) PacketTotalCount() uint64 {
	return f.MockPacketTotalCount
}

//VlanID returns which Vlan the flow took place on at the time of observation
func (f *FlowMock) VlanID() uint16 {
	return f.MockVlanID
}

//FlowEndReason returns why the metering process stopped recording the flow
func (f *FlowMock) FlowEndReason() FlowEndReason {
	return f.MockFlowEndReason
}

//Version returns the IPFIX/Netflow version
func (f *FlowMock) Version() uint8 {
	return f.MockVersion
}

//Exporter returns the address of the exporting process for this flow
func (f *FlowMock) Exporter() string {
	return f.MockExporter
}
