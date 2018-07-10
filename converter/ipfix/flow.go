package ipfix

import "github.com/activecm/ipfix-rita/converter/protocols"

//Flow represents a single IPFIX flow
type Flow interface {
	//SourceIPAddress returns the source IPv4 or IPv6 address
	SourceIPAddress() string
	//SourcePort returns the source transport port
	SourcePort() uint16
	//DestinationIPAddress returns the destination IPv4 or IPv6 address
	DestinationIPAddress() string
	//DestinationPort returns the destination transport port
	DestinationPort() uint16
	//ProtocolIdentifier returns which transport protocol was used
	ProtocolIdentifier() protocols.Identifier
	//IPClassOfService is the value of the TOS field in the IPv4 packet header or
	//the value of the Traffic Class field in the IPv6 packet header.
	IPClassOfService() uint8
	//FlowStartMilliseconds is the time the flow started as a Unix timestamp
	FlowStartMilliseconds() (int64, error)
	//FlowEndMilliseconds is the time the flow ended as a Unix timestamp
	FlowEndMilliseconds() (int64, error)
	//OctetTotalCount returns the total amount of bytes sent (including IP headers and payload)
	OctetTotalCount() int64
	//PacketTotalCount returns the number of packets sent from the source to the destination
	PacketTotalCount() int64
	//VlanID returns which Vlan the flow took place on at the time of observation
	VlanID() uint16
	//FlowEndReason returns why the metering process stopped recording the flow
	FlowEndReason() FlowEndReason
	//Version returns the IPFIX/Netflow version
	Version() uint8
	//Exporter returns the address of the exporting process for this flow
	Exporter() string
}

//FlowEndReason Represents IPFIX Information Export #136
type FlowEndReason uint8

const (
	//IdleTimeout shows the Flow was terminated because it was considered to be idle.
	IdleTimeout FlowEndReason = iota
	//ActiveTimeout shows the Flow was terminated for reporting purposes while it was
	//still active, for example, after the maximum lifetime of unreported Flows was reached.
	ActiveTimeout
	//EndOfFlow shows the Flow was terminated because the Metering Process
	//detected signals indicating the end of the Flow, for example, the TCP FIN flag.
	EndOfFlow
	//ForcedEnd shows the Flow was terminated because of some external event,
	//for example, a shutdown of the Metering Process initiated
	//by a network management application.
	ForcedEnd
	//LackOfResources shows the Flow was terminated because of lack of resources
	//available to the Metering Process and/or the Exporting Process.
	LackOfResources
	//Nil is used to represent the absence of a FlowEndReason
	Nil FlowEndReason = 255
)
