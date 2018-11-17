package mgologstash

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/pkg/errors"
	"github.com/globalsign/mgo/bson"
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

//fillFromIPFIXBSONMap reads the data from a bson map representing
//the Netflow field of Flow and inserts it into this flow,
//returning nil if the conversion was successful.
func (i *Flow) fillFromIPFIXBSONMap(ipfixMap bson.M) error {
	//First grab all the data making sure it exists in the map
	//All of these pieces of data come out as interface{}, we have
	//to recast the data back into a typed form :(
	//fmt.Println("0")
	var ok bool
	var sourceIPv4 string
	var sourceIPv6 string
	sourceIPv4Iface, sourceIPv4Ok := ipfixMap["sourceIPv4Address"]
	sourceIPv6Iface, sourceIPv6Ok := ipfixMap["sourceIPv6Address"]
	if sourceIPv4Ok {
		sourceIPv4, ok = sourceIPv4Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", sourceIPv4Iface)
		}
	} else if sourceIPv6Ok {
		sourceIPv6, ok = sourceIPv6Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", sourceIPv6Iface)
		}
	} else {
		return errors.New("input map must contain key 'netflow.sourceIPv4Address' or 'netflow.sourceIPv6Address'")
	}

	sourcePortIface, ok := ipfixMap["sourceTransportPort"]
	if !ok {
		return errors.New("input map must contain key 'netflow.sourceTransportPort'")
	}
	sourcePort, ok := sourcePortIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", sourcePortIface)
	}

	var destIPv4 string
	var destIPv6 string
	destIPv4Iface, destIPv4Ok := ipfixMap["destinationIPv4Address"]
	destIPv6Iface, destIPv6Ok := ipfixMap["destinationIPv6Address"]
	if destIPv4Ok {
		destIPv4, ok = destIPv4Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", destIPv4Iface)
		}

		postNatDestIPv4Iface, postNatDestIPv4Ok := ipfixMap["postNATDestinationIPv4Address"]

		if postNatDestIPv4Ok {
			destIPv4, ok = postNatDestIPv4Iface.(string)
			if !ok {
				return errors.Errorf("could not convert %+v to string", postNatDestIPv4Iface)
			}
		}
	} else if destIPv6Ok {
		destIPv6, ok = destIPv6Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", destIPv6Iface)
		}

		postNatDestIPv6Iface, postNatDestIPv6Ok := ipfixMap["postNATDestinationIPv6Address"]

		if postNatDestIPv6Ok {
			destIPv6, ok = postNatDestIPv6Iface.(string)
			if !ok {
				return errors.Errorf("could not convert %+v to string", postNatDestIPv6Iface)
			}
		}
	} else {
		return errors.New("input map must contain key 'netflow.destinationIPv4Address' or 'netflow.destinationIPv6Address'")
	}

	var destPort int
	destPortIface, ok := ipfixMap["destinationTransportPort"]
	if ok {

		destPort, ok = destPortIface.(int)

		if !ok {
			return errors.Errorf("could not convert %+v to int", destPortIface)
		}

		postNaptDestPortIface, postNaptDestPortIfaceOk := ipfixMap["postNAPTDestinationTransportPort"]
		if postNaptDestPortIfaceOk {
			destPort, ok = postNaptDestPortIface.(int)
			if !ok {
				return errors.Errorf("could not convert %+v to int", postNaptDestPortIface)
			}
		}

	} else {
		return errors.New("input map must contain key 'netflow.destinationTransportPort'")
	}

	flowStartIface, ok := ipfixMap["flowStartMilliseconds"]
	if !ok {
		return errors.New("input map must contain key 'netflow.flowStartMilliseconds'")
	}
	flowStart, ok := flowStartIface.(string)
	if !ok {
		return errors.Errorf("could not convert %+v to string", flowStartIface)
	}

	flowEndIface, ok := ipfixMap["flowEndMilliseconds"]
	if !ok {
		return errors.New("input map must contain key 'netflow.flowEndMilliseconds'")
	}
	flowEnd, ok := flowEndIface.(string)
	if !ok {
		return errors.Errorf("could not convert %+v to string", flowEndIface)
	}

	octetTotalIface, ok := ipfixMap["octetTotalCount"]
	if !ok {
		//delta counts CAN be total counts by RFC definition >.<"
		octetTotalIface, ok = ipfixMap["octetDeltaCount"]
		if !ok {
			return errors.New("input map must contain key 'netflow.octetTotalCount' or 'netflow.octetDeltaCount'")
		}
	}
	octetTotal, ok := octetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		octetTotal32, octetTotal32Ok := octetTotalIface.(int)
		if !octetTotal32Ok {
			return errors.Errorf("could not convert %+v to int", octetTotalIface)
		}
		octetTotal = int64(octetTotal32)
	}

	packetTotalIface, ok := ipfixMap["packetTotalCount"]
	if !ok {
		//delta counts CAN be total counts by RFC definition >.<"
		packetTotalIface, ok = ipfixMap["packetDeltaCount"]
		if !ok {
			return errors.New("input map must contain key 'netflow.packetTotalCount' or 'netflow.packetDeltaCount'")
		}
	}
	packetTotal, ok := packetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		packetTotal32, packetTotal32Ok := packetTotalIface.(int)
		if !packetTotal32Ok {
			return errors.Errorf("could not convert %+v to int", packetTotalIface)
		}
		packetTotal = int64(packetTotal32)
	}

	protocolIDIface, ok := ipfixMap["protocolIdentifier"]
	if !ok {
		return errors.New("input map must contain key 'netflow.protocolIdentifier'")
	}
	protocolID, ok := protocolIDIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", protocolIDIface)
	}

	//assume EndOfFlow if flowEndReason is not present
	flowEndReason := input.EndOfFlow
	flowEndReasonIface, ok := ipfixMap["flowEndReason"]
	if ok {
		flowEndReasonInt, flowEndReasonIntOk := flowEndReasonIface.(int)
		if !flowEndReasonIntOk {
			return errors.Errorf("could not convert %+v to int", flowEndReasonIface)
		}
		flowEndReason = input.FlowEndReason(flowEndReasonInt)
	}

	//Fill in the flow now that we know we have all the data
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
	i.Netflow.FlowEndReason = flowEndReason
	return nil
}

//fillFromNetflowv9BSONMap reads the data from a bson map representing
//the Netflow field of Flow and inserts it into this flow,
//returning nil if the conversion was successful.
func (i *Flow) fillFromNetflowv9BSONMap(netflowMap bson.M) error {
	//First grab all the data making sure it exists in the map
	//All of these pieces of data come out as interface{}, we have
	//to recast the data back into a typed form :(
	//fmt.Println("0")
	var ok bool
	var sourceIPv4 string
	var sourceIPv6 string
	sourceIPv4Iface, sourceIPv4Ok := netflowMap["ipv4_src_addr"]
	sourceIPv6Iface, sourceIPv6Ok := netflowMap["ipv6_src_addr"]
	if sourceIPv4Ok {
		sourceIPv4, ok = sourceIPv4Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", sourceIPv4Iface)
		}
	} else if sourceIPv6Ok {
		sourceIPv6, ok = sourceIPv6Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", sourceIPv6Iface)
		}
	} else {
		return errors.New("input map must contain key 'netflow.ipv4_src_addr' or 'netflow.ipv6_src_addr'")
	}

	sourcePortIface, ok := netflowMap["l4_src_port"]
	if !ok {
		return errors.New("input map must contain key 'netflow.l4_src_port'")
	}
	sourcePort, ok := sourcePortIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", sourcePortIface)
	}

	var destIPv4 string
	var destIPv6 string
	destIPv4Iface, destIPv4Ok := netflowMap["ipv4_dst_addr"]
	destIPv6Iface, destIPv6Ok := netflowMap["ipv6_dst_addr"]
	if destIPv4Ok {
		destIPv4, ok = destIPv4Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", destIPv4Iface)
		}

		postNatDestIPv4Iface, postNatDestIPv4Ok := netflowMap["xlate_dst_addr_ipv4"]
		if postNatDestIPv4Ok {
			destIPv4, ok = postNatDestIPv4Iface.(string)
			if !ok {
				return errors.Errorf("could not convert %+v to string", postNatDestIPv4Iface)
			}
		}

	} else if destIPv6Ok {
		destIPv6, ok = destIPv6Iface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", destIPv6Iface)
		}

		postNatDestIPv6Iface, postNatDestIPv6Ok := netflowMap["xlate_dst_addr_ipv6"]
		if postNatDestIPv6Ok {
			destIPv6, ok = postNatDestIPv6Iface.(string)
			if !ok {
				return errors.Errorf("could not convert %+v to string", postNatDestIPv6Iface)
			}
		}

	} else {
		return errors.New("input map must contain key 'netflow.ipv4_dst_addr' or 'netflow.ipv6_dst_addr'")
	}

	var destPort int
	destPortIface, ok := netflowMap["l4_dst_port"]
	if ok {
		destPort, ok = destPortIface.(int)
		if !ok {
			return errors.Errorf("could not convert %+v to int", destPortIface)
		}

		postNaptDestPortIface, postNatptDestPortOk := netflowMap["xlate_dst_port"]
		if postNatptDestPortOk {
			destPort, ok = postNaptDestPortIface.(int)
			if !ok {
				return errors.Errorf("could not convert %+v to int", postNaptDestPortIface)
			}
		}

	} else {
		return errors.New("input map must contain key 'netflow.l4_dst_port'")
	}

	flowStartIface, ok := netflowMap["first_switched"]
	if !ok {
		return errors.New("input map must contain key 'netflow.first_switched'")
	}
	flowStart, ok := flowStartIface.(string)
	if !ok {
		return errors.Errorf("could not convert %+v to string", flowStartIface)
	}

	flowEndIface, ok := netflowMap["last_switched"]
	if !ok {
		return errors.New("input map must contain key 'netflow.last_switched'")
	}
	flowEnd, ok := flowEndIface.(string)
	if !ok {
		return errors.Errorf("could not convert %+v to string", flowEndIface)
	}

	octetTotalIface, ok := netflowMap["in_bytes"]
	if !ok {
		return errors.New("input map must contain key 'netflow.in_bytes'")
	}
	octetTotal, ok := octetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		octetTotal32, octetTotal32Ok := octetTotalIface.(int)
		if !octetTotal32Ok {
			return errors.Errorf("could not convert %+v to int", octetTotalIface)
		}
		octetTotal = int64(octetTotal32)
	}

	packetTotalIface, ok := netflowMap["in_pkts"]
	if !ok {
		return errors.New("input map must contain key 'netflow.in_pkts'")
	}
	packetTotal, ok := packetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		packetTotal32, packetTotal32Ok := packetTotalIface.(int)
		if !packetTotal32Ok {
			return errors.Errorf("could not convert %+v to int", packetTotalIface)
		}
		packetTotal = int64(packetTotal32)
	}

	protocolIDIface, ok := netflowMap["protocol"]
	if !ok {
		return errors.New("input map must contain key 'netflow.protocol'")
	}
	protocolID, ok := protocolIDIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", protocolIDIface)
	}

	//Fill in the flow now that we know we have all the data
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
	//assume end of flow since we don't have the data
	i.Netflow.FlowEndReason = input.EndOfFlow
	return nil
}

//fillFromNetflowv5BSONMap reads the data from a bson map representing
//the Netflow field of Flow and inserts it into this flow,
//returning nil if the conversion was successful.
func (i *Flow) fillFromNetflowv5BSONMap(netflowMap bson.M) error {
	//First grab all the data making sure it exists in the map
	//All of these pieces of data come out as interface{}, we have
	//to recast the data back into a typed form :(
	//fmt.Println("0")
	var ok bool
	var sourceIPv4 string
	sourceIPIface, sourceIPOk := netflowMap["srcaddr"]
	if sourceIPOk {
		sourceIPv4, ok = sourceIPIface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", sourceIPIface)
		}
	} else {
		return errors.New("input map must contain key 'netflow.ipv4_src_addr' or 'netflow.ipv6_src_addr'")
	}

	sourcePortIface, ok := netflowMap["srcport"]
	if !ok {
		return errors.New("input map must contain key 'netflow.l4_src_port'")
	}
	sourcePort, ok := sourcePortIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", sourcePortIface)
	}

	var destIP string
	destIPIface, destIPOk := netflowMap["dstaddr"]
	if destIPOk {
		destIP, ok = destIPIface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", destIPIface)
		}
	} else {
		return errors.New("input map must contain key 'netflow.dstaddr'")
	}

	var destPort int
	destPortIface, ok := netflowMap["dstport"]
	if ok {
		destPort, ok = destPortIface.(int)
		if !ok {
			return errors.Errorf("could not convert %+v to int", destPortIface)
		}
	} else {
		return errors.New("input map must contain key 'netflow.dstport'")
	}

	flowStartIface, ok := netflowMap["first"]
	if !ok {
		return errors.New("input map must contain key 'netflow.first'")
	}
	flowStart, ok := flowStartIface.(string)
	if !ok {
		return errors.Errorf("could not convert %+v to string", flowStartIface)
	}

	flowEndIface, ok := netflowMap["last"]
	if !ok {
		return errors.New("input map must contain key 'netflow.last'")
	}
	flowEnd, ok := flowEndIface.(string)
	if !ok {
		return errors.Errorf("could not convert %+v to string", flowEndIface)
	}

	octetTotalIface, ok := netflowMap["dOctets"]
	if !ok {
		return errors.New("input map must contain key 'netflow.dOctets'")
	}
	octetTotal, ok := octetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		octetTotal32, octetTotal32Ok := octetTotalIface.(int)
		if !octetTotal32Ok {
			return errors.Errorf("could not convert %+v to int", octetTotalIface)
		}
		octetTotal = int64(octetTotal32)
	}

	packetTotalIface, ok := netflowMap["dPkts"]
	if !ok {
		return errors.New("input map must contain key 'netflow.dPkts'")
	}
	packetTotal, ok := packetTotalIface.(int64)
	if !ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		packetTotal32, packetTotal32Ok := packetTotalIface.(int)
		if !packetTotal32Ok {
			return errors.Errorf("could not convert %+v to int", packetTotalIface)
		}
		packetTotal = int64(packetTotal32)
	}

	protocolIDIface, ok := netflowMap["prot"]
	if !ok {
		return errors.New("input map must contain key 'netflow.prot'")
	}
	protocolID, ok := protocolIDIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", protocolIDIface)
	}

	//Fill in the flow now that we know we have all the data
	if sourceIPv4Ok {
		i.Netflow.SourceIPv4 = sourceIPv4
	}

	i.Netflow.SourcePort = uint16(sourcePort)

	if destIPv4Ok {
		i.Netflow.DestinationIPv4 = destIPv4
	}

	i.Netflow.DestinationPort = uint16(destPort)

	i.Netflow.FlowStartMilliseconds = flowStart
	i.Netflow.FlowEndMilliseconds = flowEnd
	i.Netflow.OctetTotalCount = octetTotal
	i.Netflow.PacketTotalCount = packetTotal
	i.Netflow.ProtocolIdentifier = protocols.Identifier(protocolID)
	//assume end of flow since we don't have the data
	i.Netflow.FlowEndReason = input.EndOfFlow
	return nil
}

//FillFromBSONMap reads the data from a bson map and inserts
//it into this flow, returning nil if the conversion was successful.
//This method is used for filtering input data and adapting
//multiple versions of netflow records to the same data type.
func (i *Flow) FillFromBSONMap(inputMap bson.M) error {
	idIface, ok := inputMap["_id"]
	if !ok {
		return errors.New("input map must contain key '_id'")
	}
	id, ok := idIface.(bson.ObjectId)
	if !ok {
		return errors.Errorf("could not convert %+v to bson.ObjectID", idIface)
	}

	hostIface, ok := inputMap["host"]
	if !ok {
		return errors.New("input map must contain key 'host'")
	}
	host, ok := hostIface.(string)
	if !ok {
		return errors.Errorf("could not convert %+v to string", hostIface)
	}

	netflowMapIface, ok := inputMap["netflow"]
	if !ok {
		return errors.New("input map must contain key 'netflow'")
	}
	netflowMap, ok := netflowMapIface.(bson.M)
	if !ok {
		return errors.Errorf("could not convert %+v to bson.M", netflowMapIface)
	}

	versionIface, ok := netflowMap["version"]
	if !ok {
		return errors.New("input map must contain key 'netflow.version'")
	}
	version, ok := versionIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", versionIface)
	}

	//set the loaded contents
	i.ID = id
	i.Host = host
	i.Netflow.Version = uint8(version)

	//Version must be 10 or 9
	if i.Netflow.Version == 10 {
		return i.fillFromIPFIXBSONMap(netflowMap)
	} else if i.Netflow.Version == 9 {
		return i.fillFromNetflowv9BSONMap(netflowMap)
	} else if i.Netflow.Version == 5 {
		return i.fillFromNetflowv5BSONMap(netflowMap)
	}
	return errors.Errorf("unsupported netflow version: %d", i.Netflow.Version)
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
