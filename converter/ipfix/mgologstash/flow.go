package mgologstash

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

//Flow represents an IPFIX flow record stored in MongoDB via Logstash
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
		IPClassOfService   uint8                `bson:"ipClassOfService"`
		VlanID             uint16               `bson:"vlanId"`
		FlowEndReason      ipfix.FlowEndReason  `bson:"flowEndReason"`
		Version            uint8                `bson:"version"`
	} `bson:"netflow"`
}

//FillFromMgoMap reads the data from a bson map and inserts
//it into this flow, returning true if the conversion was successful.
//This method is used for filtering input data. Otherwise,
//the data could be read directly into the struct with mgo.
func (i *Flow) FillFromMgoMap(inputMap bson.M) bool {
	//First grab all the data making sure it exists in the map
	//All of these pieces of data come out as interface{}, we have
	//to recast the data back into a typed form :(
	//fmt.Println("0")
	idIface, ok := inputMap["_id"]
	if !ok {
		return false
	}
	//fmt.Println("1")
	id, ok := idIface.(bson.ObjectId)
	if !ok {
		return false
	}
	//fmt.Println("2")

	hostIface, ok := inputMap["host"]
	if !ok {
		return false
	}
	//fmt.Println("3")

	host, ok := hostIface.(string)
	if !ok {
		return false
	}
	//fmt.Println("4")

	netflowMapIface, ok := inputMap["netflow"]
	if !ok {
		return false
	}
	//fmt.Println("5")

	netflowMap, ok := netflowMapIface.(bson.M)
	if !ok {
		return false
	}
	//fmt.Println("6")

	var sourceIPv4 string
	var sourceIPv6 string
	sourceIPv4Iface, sourceIPv4Ok := netflowMap["sourceIPv4Address"]
	sourceIPv6Iface, sourceIPv6Ok := netflowMap["sourceIPv6Address"]
	if sourceIPv4Ok {
		//fmt.Println("7")
		sourceIPv4, ok = sourceIPv4Iface.(string)
		if !ok {
			return false
		}
	} else if sourceIPv6Ok {
		//fmt.Println("8")
		sourceIPv6, ok = sourceIPv6Iface.(string)
		if !ok {
			return false
		}
	} else {
		//fmt.Println("9")
		return false
	}
	//fmt.Println("10")

	sourcePortIface, ok := netflowMap["sourceTransportPort"]
	if !ok {
		return false
	}
	//fmt.Println("11")
	sourcePort, ok := sourcePortIface.(int)
	if !ok {
		return false
	}
	//fmt.Println("12")

	var destIPv4 string
	var destIPv6 string
	destIPv4Iface, destIPv4Ok := netflowMap["destinationIPv4Address"]
	destIPv6Iface, destIPv6Ok := netflowMap["destinationIPv6Address"]
	if destIPv4Ok {
		//fmt.Println("13")
		destIPv4, ok = destIPv4Iface.(string)
		if !ok {
			return false
		}
	} else if destIPv6Ok {
		//fmt.Println("14")
		destIPv6, ok = destIPv6Iface.(string)
		if !ok {
			return false
		}
	} else {
		//fmt.Println("15")
		return false
	}
	//fmt.Println("16")

	destPortIface, ok := netflowMap["destinationTransportPort"]
	if !ok {
		return false
	}
	//fmt.Println("17")
	destPort, ok := destPortIface.(int)
	if !ok {
		return false
	}
	//fmt.Println("18")

	flowStartIface, ok := netflowMap["flowStartMilliseconds"]
	if !ok {
		return false
	}
	//fmt.Println("19")
	flowStart, ok := flowStartIface.(string)
	if !ok {
		return false
	}
	//fmt.Println("20")

	flowEndIface, ok := netflowMap["flowEndMilliseconds"]
	if !ok {
		return false
	}
	//fmt.Println("21")
	flowEnd, ok := flowEndIface.(string)
	if !ok {
		return false
	}
	//fmt.Println("22")

	octetTotalIface, ok := netflowMap["octetTotalCount"]
	if !ok {
		return false
	}
	//fmt.Println("23")
	octetTotal, ok := octetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		octetTotal32, ok := octetTotalIface.(int)
		if !ok {
			return false
		}
		octetTotal = int64(octetTotal32)
	}
	//fmt.Println("24")

	packetTotalIface, ok := netflowMap["packetTotalCount"]
	if !ok {
		return false
	}
	//fmt.Println("25")
	packetTotal, ok := packetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		packetTotal32, ok := packetTotalIface.(int)
		if !ok {
			return false
		}
		packetTotal = int64(packetTotal32)
	}
	//fmt.Println("26")

	protocolIDIface, ok := netflowMap["protocolIdentifier"]
	if !ok {
		return false
	}
	//fmt.Println("27")
	protocolID, ok := protocolIDIface.(int)
	if !ok {
		return false
	}
	//fmt.Println("28")

	ipClassOfServiceIface, ok := netflowMap["ipClassOfService"]
	if !ok {
		return false
	}
	//fmt.Println("29")
	ipClassOfService, ok := ipClassOfServiceIface.(int)
	if !ok {
		return false
	}
	//fmt.Println("30")

	vlanIDIface, ok := netflowMap["vlanId"]
	if !ok {
		return false
	}
	//fmt.Println("31")
	vlanID, ok := vlanIDIface.(int)
	if !ok {
		return false
	}
	//fmt.Println("32")

	flowEndReasonIface, ok := netflowMap["flowEndReason"]
	if !ok {
		return false
	}
	//fmt.Println("33")
	flowEndReason, ok := flowEndReasonIface.(int)
	if !ok {
		return false
	}
	//fmt.Println("34")

	versionIface, ok := netflowMap["version"]
	if !ok {
		return false
	}
	//fmt.Println("35")
	version, ok := versionIface.(int)
	if !ok {
		return false
	}
	//fmt.Println("36")

	//Fill in the flow now that we know we have all the data
	i.ID = id
	i.Host = host
	if sourceIPv4Ok {
		i.Netflow.SourceIPv4 = sourceIPv4
	}
	if sourceIPv6Ok {
		i.Netflow.SourceIPv6 = sourceIPv6
	}

	i.Netflow.SourcePort = uint16(sourcePort)

	if destIPv4Ok {
		i.Netflow.DestinationIPv4 = destIPv4
	}
	if destIPv6Ok {
		i.Netflow.DestinationIPv6 = destIPv6
	}

	i.Netflow.DestinationPort = uint16(destPort)

	i.Netflow.FlowStartMilliseconds = flowStart
	i.Netflow.FlowEndMilliseconds = flowEnd
	i.Netflow.OctetTotalCount = octetTotal
	i.Netflow.PacketTotalCount = packetTotal
	i.Netflow.ProtocolIdentifier = protocols.Identifier(protocolID)
	i.Netflow.IPClassOfService = uint8(ipClassOfService)
	i.Netflow.VlanID = uint16(vlanID)
	i.Netflow.FlowEndReason = ipfix.FlowEndReason(flowEndReason)
	i.Netflow.Version = uint8(version)
	return true
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

//IPClassOfService is the value of the TOS field in the IPv4 packet header or
//the value of the Traffic Class field in the IPv6 packet header.
func (i *Flow) IPClassOfService() uint8 {
	return i.Netflow.IPClassOfService
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

//VlanID returns which Vlan the flow took place on at the time of observation
func (i *Flow) VlanID() uint16 {
	return i.Netflow.VlanID
}

//FlowEndReason returns why the metering process stopped recording the flow
func (i *Flow) FlowEndReason() ipfix.FlowEndReason {
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
