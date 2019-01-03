package mgologstash

import (
	"time"
	// "fmt"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

type ipfixRelTime struct {
	firstFlowMills int64
	firstFlowTime  time.Time
	firstFlowSet   bool
}

//flowDeserializer converts a sequence of IPFIX/Netflow v5/v9 ,
//Logstash created, BSON maps into mgologstash.Flow objects.
//The deserializer encapsulates the deserialization methods
//for IPFIX. Netflow v9, and Netflow v5.
//Netflow v9 and v5 deserialization may as well be implemented as static
//methods as they do not require state. They are not implemented as static
//methods, however, in order to match the way IPFIX deserialization is implemented.
//IPFIX deserialization requires state to be held while IPFIX
//records are deserialized. This is due to the fact that IPFIX allows
//timestamps to be represented as system boot time relative values.
//The system boot time (systemInitTimeMilliseconds) may be sent in a
//IPFIX record other than the record to be processed. As such,
//the system boot time for each exporting host must be held as state
//while sequences of IPFIX records are deserialized.

type flowDeserializer struct {
	ipfixExporterUptimes map[string]int64 //map from exporting host to systemInitTimeMilliseconds values
	ipfixExporterRelUp   map[string]ipfixRelTime //map from exporting host to relative system uptime values
}

func newFlowDeserializer() *flowDeserializer {
	return &flowDeserializer{
		ipfixExporterUptimes: make(map[string]int64),
		ipfixExporterRelUp: make(map[string]ipfixRelTime),
	}
}

//updateExporterUptimesMap updates the host entry in the ipfixExporterUptimes map
//if the ipfixMap contains a systemInitTimeMilliseconds field.
//If the update is successful, the function returns true. Otherwise
//the function returns false.
func (f *flowDeserializer) updateExporterUptimesMap(ipfixMap bson.M, host string) bool {
	//update the ipfixExporterUptimes map if the data is available
	exporterUptimeIface, exporterUptimeOk := ipfixMap["systemInitTimeMilliseconds"]
	if exporterUptimeOk {
		var exporterUptime int64
		exporterUptime, exporterUptimeOk = exporterUptimeIface.(int64)
		if !exporterUptimeOk {
			//Logstash creates these fields as 32 bit ints,
			//Go handles them as 64 bit ints, provide both casts
			var exporterUptime32 int
			exporterUptime32, exporterUptimeOk = exporterUptimeIface.(int)
			if exporterUptimeOk {
				exporterUptime = int64(exporterUptime32)
			}
		}

		if exporterUptimeOk {
			//update the map
			f.ipfixExporterUptimes[host] = exporterUptime
			return true
		}
	}
	return false
}

//updateExporterTimestamps will update the relative timestamps for each host
//relative to the daily first flow, so if we don't have an instance of the
//system init time we can still get results from RITA
func (f *flowDeserializer) updateExporterTimestamps(ipfixMap bson.M, host string) bool {
	//if we have a inital set value see if we need to update the value
	if f.ipfixExporterRelUp[host].firstFlowSet == true {
		//if the system has reinitialized then the relative timestamps will be off
		//  as a result check if there is a change and update it if needed
		//get the timestamp value so we have the full picture
		var startMills int64
		startMillsIface, startMillsOk := ipfixMap["flowStartSysUpTime"]
		if !startMillsOk {
			return false
		}
		//try converting to an int64 first, handle any errors that come up
		startMills, startMillsOk = startMillsIface.(int64)
		if !startMillsOk {
			startMills32, start32Ok := startMillsIface.(int)
			if !start32Ok {
				return false
			}
			startMills = int64(startMills32)
		}

		//if the host's first flow milliseconds is greater than the new flow's
		//  start milliseconds it implies that the system was reinitialized and it
		//  is imparitive to update the information currently saved
		if f.ipfixExporterRelUp[host].firstFlowMills > startMills {
			newExporter, err := getNewExporterUptime(ipfixMap)
			if err != nil {
				return false
			}

			f.ipfixExporterRelUp[host] = newExporter
			return true
		}
	} else {
		newExporter, err := getNewExporterUptime(ipfixMap)
		if err != nil {
			return false
		}
		f.ipfixExporterRelUp[host] = newExporter

		return true
	}

	return false
}

//Since the code for updating the relative system uptime and creating a new
//relative system uptime are similar make a function so we don't repeat code
//It returns a structure defined as ipfixRelTime (relative uptime) and any error
// this code experiences
func getNewExporterUptime(ipfixMap bson.M) (ipfixRelTime, error) {
	emptyRelTime := ipfixRelTime{0.0, time.Now(), false}

	//set the inital value if it hasn't been set
	var startMills int64
	startMillsIface, startMillsOk := ipfixMap["flowStartSysUpTime"]
	//if the milliseconds since the init isn't present return false
	if !startMillsOk {
		return emptyRelTime, errors.Errorf("Couldn't find flowStartSysUpTime")
	}
	//try converting to an int64 first, handle any errors that come up
	startMills, startMillsOk = startMillsIface.(int64)
	if !startMillsOk {
		startMills32, start32Ok := startMillsIface.(int)
		if !start32Ok {
			return emptyRelTime, errors.Errorf("Couldn't convert values")
		}
		startMills = int64(startMills32)
	}

	//get the timestamp value, if we can, then convert it to a time value
	var flowDate time.Time
	flowDateIface, flowDateOk := ipfixMap["timestamp"]
	if flowDateOk {
		//convert the time to a string, so we can parse the string to a date/time vlaue
		flowDateStr, ok := flowDateIface.(string)
		if !ok {
			return emptyRelTime, errors.Errorf("Couldn't convert timestamp to string")
		}
		var err error
		flowDate, err = time.Parse(time.RFC3339, flowDateStr)
		if err != nil {
			return emptyRelTime, err
		}
	} else {
		//if we can't find the timestamp value assume the flow came now and use
		//  that value
		flowDate = time.Now()
	}

	return ipfixRelTime{startMills, flowDate, true}, nil
}

//getFlowUptimeMillis64 will take a flow uptime (milliseconds) as an interface
//  and do the conversion to get it as an int64 for calculations
func getFlowUptimeMillis64(flowUptimeMillisIface interface{}) (int64, error) {
	flowUptimeMillis64, flowUptimeMillis64Ok := (flowUptimeMillisIface).(int64)
	if !flowUptimeMillis64Ok {
		//Logstash creates these fields as 32 bit ints,
		//Go handles them as 64 bit ints, provide both casts
		flowUptimeMillis32, flowUptimeMillis32Ok := (flowUptimeMillisIface).(int)
		if !flowUptimeMillis32Ok {
			return 0, errors.Errorf("could not convert %+v to int", flowUptimeMillisIface)
		}
		flowUptimeMillis64 = int64(flowUptimeMillis32)
	}

	return flowUptimeMillis64, nil
}

//fillFromIPFIXBSONMap reads the data from a bson map representing
//the Netflow field of Flow and inserts it into this flow,
//returning nil if the conversion was successful.
//The exporting host must be provided in order to resolve flowStartSysUpTime and
//flowEndSysUpTime timestamps.
func (f *flowDeserializer) fillFromIPFIXBSONMap(ipfixMap bson.M, outputFlow *Flow, host string) error {
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

	var flowStart, flowEnd string // RFC3339Nano timestamps

	//Get the Start, and the end times of the flow
	flowStartMillisIface, flowStartMillisOk := ipfixMap["flowStartMilliseconds"]
	flowEndMillisIface, flowEndMillisOk := ipfixMap["flowEndMilliseconds"]

	//Also attempt to get the start and end times relative to system init
	flowStartUptimeMillisIface, flowStartUptimeMillisOk := ipfixMap["flowStartSysUpTime"]
	flowEndUptimeMillisIface, flowEndUptimeMillisOk := ipfixMap["flowEndSysUpTime"]

	//get the system init time if possible
	systemInitTimeMillis, systemInitTimeMillisecondsOk := f.ipfixExporterUptimes[host]
	//If the system init time isn't present or stored use a relative uptime approach
	systemRelativeMillis, systemRelativeOk := f.ipfixExporterRelUp[host]

	if flowStartMillisOk && flowEndMillisOk {
		//Case 1: We have an absolute start and end time (this is ideal)
		flowStart, flowStartMillisOk = flowStartMillisIface.(string)
		if !flowStartMillisOk {
			return errors.Errorf("could not convert %+v to string", flowStartMillisIface)
		}
		flowEnd, flowEndMillisOk = flowEndMillisIface.(string)
		if !flowEndMillisOk {
			return errors.Errorf("could not convert %+v to string", flowEndMillisIface)
		}
	} else if flowStartUptimeMillisOk && flowEndUptimeMillisOk && systemInitTimeMillisecondsOk {
		//Case 2: We have an start and end time in milliseconds from system init and
		//  we have the absolute system init time, we can find the absolute start
		//  and end time of each flow, less ideal because of computation time
		flowStartUptimeMillis64, convErr := getFlowUptimeMillis64(flowStartUptimeMillisIface)
		if convErr != nil {
			return convErr
		}

		flowEndUptimeMillis64, convErr := getFlowUptimeMillis64(flowEndUptimeMillisIface)
		if convErr != nil {
			return convErr
		}

		flowStartUnixMillis := systemInitTimeMillis + flowStartUptimeMillis64
		flowStartUnixSeconds := flowStartUnixMillis / 1000
		flowStartUnixNanos := (flowStartUnixMillis % 1000) * 1000000

		flowEndUnixMillis := systemInitTimeMillis + flowEndUptimeMillis64
		flowEndUnixSeconds := flowEndUnixMillis / 1000
		flowEndUnixNanos := (flowEndUnixMillis % 1000) * 1000000

		flowStart = time.Unix(flowStartUnixSeconds, flowStartUnixNanos).Format(time.RFC3339Nano)
		flowEnd = time.Unix(flowEndUnixSeconds, flowEndUnixNanos).Format(time.RFC3339Nano)
	} else if flowStartUptimeMillisOk && flowEndUptimeMillisOk && systemRelativeOk {
		//Case 3: We have an start and end time in milliseconds from system init and
		//  we have a timestamp from when the first flow was made available, less
		//  ideal yet as the time of flow will be off but the beaconing algorithm
		//  still works
		flowStartUptimeMillis64, convErr := getFlowUptimeMillis64(flowStartUptimeMillisIface)
		if convErr != nil {
			return convErr
		}

		flowEndUptimeMillis64, convErr := getFlowUptimeMillis64(flowEndUptimeMillisIface)
		if convErr != nil {
			return convErr
		}

		firstFlowTime := systemRelativeMillis.firstFlowTime
		firstFlowMillis := systemRelativeMillis.firstFlowMills

		//By finding the difference in time from the first flow and the new flow
		//  start and end we can calculate the time since the first flow and add that
		//  difference to the time of the first flow
		//Note: since the time.Add function takes a time.Duration (which is a int64
	  //  nanoseconds count) we need to convert time to nanoseconds from ms
		//  this is done by miltiplying by 1000000
		flowStartOffsetNanos := (flowStartUptimeMillis64 - firstFlowMillis) * 1000000
		flowEndOffsetNanos := (flowEndUptimeMillis64 - firstFlowMillis) * 1000000

		flowStartTime := firstFlowTime.Add(time.Duration(flowStartOffsetNanos))
		flowEndTime := firstFlowTime.Add(time.Duration(flowEndOffsetNanos))

		flowStart = flowStartTime.Format(time.RFC3339Nano)
		flowEnd = flowEndTime.Format(time.RFC3339Nano)
	} else {
		//Case 4: We have no timing information, all hope is lost, and may God have
		//  mercy on our souls...
		//  https://youtu.be/lpZiPZwwXhM
		return errors.New(
			"input must contain valid start and end timestamps.\n\n" +
				"If this problem persists, please report this problem at\n" +
				"support@activecountermeasures.com. If your device supports\n" +
				"alternative versions of Netflow, you may resolve this issue by\n" +
				"disabling IPFIX and enabling Netflow version 5 or 9. ",
		)
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
		outputFlow.Netflow.SourceIPv4 = sourceIPv4
	}
	if sourceIPv6Ok {
		outputFlow.Netflow.SourceIPv6 = sourceIPv6
	}

	outputFlow.Netflow.SourcePort = uint16(sourcePort)

	if destIPv4Ok {
		outputFlow.Netflow.DestinationIPv4 = destIPv4
	}
	if destIPv6Ok {
		outputFlow.Netflow.DestinationIPv6 = destIPv6
	}

	outputFlow.Netflow.DestinationPort = uint16(destPort)

	outputFlow.Netflow.FlowStartMilliseconds = flowStart
	outputFlow.Netflow.FlowEndMilliseconds = flowEnd
	outputFlow.Netflow.OctetTotalCount = octetTotal
	outputFlow.Netflow.PacketTotalCount = packetTotal
	outputFlow.Netflow.ProtocolIdentifier = protocols.Identifier(protocolID)
	outputFlow.Netflow.FlowEndReason = flowEndReason
	return nil
}

//fillFromNetflowv9BSONMap reads the data from a bson map representing
//the Netflow field of Flow and inserts it into this flow,
//returning nil if the conversion was successful.
func (f *flowDeserializer) fillFromNetflowv9BSONMap(netflowMap bson.M, outputFlow *Flow) error {
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
		outputFlow.Netflow.SourceIPv4 = sourceIPv4
	}
	if sourceIPv6Ok {
		outputFlow.Netflow.SourceIPv6 = sourceIPv6
	}

	outputFlow.Netflow.SourcePort = uint16(sourcePort)

	if destIPv4Ok {
		outputFlow.Netflow.DestinationIPv4 = destIPv4
	}
	if destIPv6Ok {
		outputFlow.Netflow.DestinationIPv6 = destIPv6
	}

	outputFlow.Netflow.DestinationPort = uint16(destPort)

	outputFlow.Netflow.FlowStartMilliseconds = flowStart
	outputFlow.Netflow.FlowEndMilliseconds = flowEnd
	outputFlow.Netflow.OctetTotalCount = octetTotal
	outputFlow.Netflow.PacketTotalCount = packetTotal
	outputFlow.Netflow.ProtocolIdentifier = protocols.Identifier(protocolID)
	//assume end of flow since we don't have the data
	outputFlow.Netflow.FlowEndReason = input.EndOfFlow
	return nil
}

//fillFromNetflowv5BSONMap reads the data from a bson map representing
//the Netflow field of Flow and inserts it into this flow,
//returning nil if the conversion was successful.
func (f *flowDeserializer) fillFromNetflowv5BSONMap(netflowMap bson.M, outputFlow *Flow) error {
	//First grab all the data making sure it exists in the map
	//All of these pieces of data come out as interface{}, we have
	//to recast the data back into a typed form :(
	//fmt.Println("0")
	var ok bool
	var sourceIP string
	sourceIPIface, sourceIPOk := netflowMap["ipv4_src_addr"]
	if sourceIPOk {
		sourceIP, ok = sourceIPIface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", sourceIPIface)
		}
	} else {
		return errors.New("input map must contain key 'netflow.ipv4_src_addr'")
	}

	sourcePortIface, ok := netflowMap["l4_src_port"]
	if !ok {
		return errors.New("input map must contain key 'netflow.l4_src_port'")
	}
	sourcePort, ok := sourcePortIface.(int)
	if !ok {
		return errors.Errorf("could not convert %+v to int", sourcePortIface)
	}

	var destIP string
	destIPIface, destIPOk := netflowMap["ipv4_dst_addr"]
	if destIPOk {
		destIP, ok = destIPIface.(string)
		if !ok {
			return errors.Errorf("could not convert %+v to string", destIPIface)
		}
	} else {
		return errors.New("input map must contain key 'netflow.ipv4_dst_addr'")
	}

	var destPort int
	destPortIface, ok := netflowMap["l4_dst_port"]
	if ok {
		destPort, ok = destPortIface.(int)
		if !ok {
			return errors.Errorf("could not convert %+v to int", destPortIface)
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
	outputFlow.Netflow.SourceIPv4 = sourceIP
	outputFlow.Netflow.SourcePort = uint16(sourcePort)

	outputFlow.Netflow.DestinationIPv4 = destIP
	outputFlow.Netflow.DestinationPort = uint16(destPort)

	outputFlow.Netflow.FlowStartMilliseconds = flowStart
	outputFlow.Netflow.FlowEndMilliseconds = flowEnd
	outputFlow.Netflow.OctetTotalCount = octetTotal
	outputFlow.Netflow.PacketTotalCount = packetTotal
	outputFlow.Netflow.ProtocolIdentifier = protocols.Identifier(protocolID)
	//assume end of flow since we don't have the data
	outputFlow.Netflow.FlowEndReason = input.EndOfFlow
	return nil
}

//deserializeNextBSONMap reads the data from a bson map and inserts
//it into the output flow, returning nil if the conversion was successful.
//This method is used for filtering input data and adapting
//multiple versions of netflow records to the same data type.
//If the inputMap contains data that must be maintained as state,
//for example, IPFIX's systemInitTimeMilliseconds, the state will be retained
//even if the flow is only partially filled and an error is returned.
func (f *flowDeserializer) deserializeNextBSONMap(inputMap bson.M, outputFlow *Flow) error {
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
	outputFlow.ID = id
	outputFlow.Host = host
	outputFlow.Netflow.Version = uint8(version)

	//Version must be 10 or 9
	if outputFlow.Netflow.Version == 10 {
		//handle recording systemInitTimeMilliseconds
		f.updateExporterUptimesMap(netflowMap, host)
		//theres a chance that systemInitTimeMilliseconds
		//came inside a flow record, parse the rest out just in case...
		//unfortunately, we can't tell option records from flow records
		f.updateExporterTimestamps(netflowMap, host)

		return f.fillFromIPFIXBSONMap(netflowMap, outputFlow, host)
	} else if outputFlow.Netflow.Version == 9 {
		return f.fillFromNetflowv9BSONMap(netflowMap, outputFlow)
	} else if outputFlow.Netflow.Version == 5 {
		return f.fillFromNetflowv5BSONMap(netflowMap, outputFlow)
	}
	return errors.Errorf("unsupported netflow version: %d", outputFlow.Netflow.Version)
}
