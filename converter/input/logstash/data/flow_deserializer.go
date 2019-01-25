package data

import (
	"time"
	// "fmt"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/input/logstash/data/safemap"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/pkg/errors"
)

type ipfixRelTime struct {
	firstFlowMills int64
	firstFlowTime  time.Time
}

//FlowDeserializer converts a sequence of IPFIX/Netflow v5/v9 ,
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
type FlowDeserializer struct {
	ipfixExporterAbsUptimes map[string]int64        //map from exporting host to systemInitTimeMilliseconds values
	ipfixExporterRelUptimes map[string]ipfixRelTime //map from exporting host to relative system uptime values
}

//NewFlowDeserializer creates a new FlowDeserializer
func NewFlowDeserializer() *FlowDeserializer {
	return &FlowDeserializer{
		ipfixExporterAbsUptimes: make(map[string]int64),
		ipfixExporterRelUptimes: make(map[string]ipfixRelTime),
	}
}

//updateExporterAbsUptimes updates the host entry in the ipfixExporterAbsUptimes map
//if the ipfixMap contains a systemInitTimeMilliseconds field.
//If the update is successful, the function returns true. Otherwise
//the function returns false.
func (f *FlowDeserializer) updateExporterAbsUptimes(ipfixMap safemap.SafeMap, host string) error {
	exporterUptime, err := ipfixMap.GetIntAsInt64("systemInitTimeMilliseconds")
	//update the ipfixExporterAbsUptimes map if the data is available
	if err == nil {
		f.ipfixExporterAbsUptimes[host] = exporterUptime
		return nil
	}
	return errors.Wrap(err, "input map should contain an int value for netflow.systemInitTimeMilliseconds")
}

//updateExporterRelUptimes will update the relative timestamps for each host
//relative to the daily first flow, so if we don't have an instance of the
//system init time we can still get results from RITA
func (f *FlowDeserializer) updateExporterRelUptimes(ipfixMap safemap.SafeMap, host string) error {
	//if we have a inital set value see if we need to update the value
	relUptime, ok := f.ipfixExporterRelUptimes[host]
	if ok {
		//if the system has reinitialized then the relative timestamps will be off
		//as a result check if there is a change and update it if needed
		//get the timestamp value so we have the full picture
		endMillis, err := ipfixMap.GetIntAsInt64("flowEndSysUpTime")
		if err != nil {
			return errors.Wrap(err, "input map should contain an int value for 'netflow.flowEndSysUpTime'")
		}

		//if the host's first flow milliseconds is greater than the new flow's
		//  start milliseconds it implies that the system was reinitialized and it
		//  is imparitive to update the information currently saved
		if relUptime.firstFlowMills > endMillis {
			newExporter, err := getNewExporterUptime(ipfixMap)
			if err != nil {
				return err
			}

			f.ipfixExporterRelUptimes[host] = newExporter
			return nil
		}
	}

	// If we haven't found the host in the Relative uptime map, create it
	newExporter, err := getNewExporterUptime(ipfixMap)
	if err != nil {
		return err
	}

	//assign a new rel uptime for the host
	f.ipfixExporterRelUptimes[host] = newExporter

	return nil
}

//Since the code for updating the relative system uptime and creating a new
//relative system uptime are similar make a function so we don't repeat code
//It returns a structure defined as ipfixRelTime (relative uptime) and any error
// this code experiences
func getNewExporterUptime(ipfixMap safemap.SafeMap) (ipfixRelTime, error) {
	emptyRelTime := ipfixRelTime{0.0, time.Now()}

	endMillis, err := ipfixMap.GetIntAsInt64("flowEndSysUpTime")
	if err != nil {
		return emptyRelTime, errors.Wrap(err, "input map should contain an int value for 'netflow.flowEndSysUpTime'")
	}

	//get the timestamp value, if we can, then convert it to a time value
	var flowDate time.Time
	flowDateStr, err := ipfixMap.GetString("timestamp")
	if err == nil {
		flowDate, err = time.Parse(time.RFC3339, flowDateStr)
		if err != nil {
			return emptyRelTime, errors.Wrap(err, "input map timestamp should be RFC3339")
		}
	} else {
		//if we can't find the timestamp value assume the flow came now and use
		//  that value
		flowDate = time.Now()
	}

	return ipfixRelTime{endMillis, flowDate}, nil
}

//fillFromIPFIXBSONMap reads the data from a bson map representing
//the Netflow field of Flow and inserts it into this flow,
//returning nil if the conversion was successful.
//The exporting host must be provided in order to resolve flowStartSysUpTime and
//flowEndSysUpTime timestamps.
func (f *FlowDeserializer) fillFromIPFIXBSONMap(ipfixMap safemap.SafeMap, outputFlow *Flow, host string) error {
	//First grab all the data making sure it exists in the map
	//All of these pieces of data come out as interface{}, we have
	//to recast the data back into a typed form :(
	//fmt.Println("0")
	var ok bool
	sourceIPv4, srcIPv4err := ipfixMap.GetString("sourceIPv4Address")
	sourceIPv6, srcIPv6err := ipfixMap.GetString("sourceIPv6Address")
	if srcIPv4err != nil && srcIPv6err != nil {
		return errors.Wrapf(
			srcIPv4err, "input map must contain a string value for "+
				"'netflow.sourceIPv4Address' or 'netflow.sourceIPv6Address'\n"+
				"Additional Cause:\n %+v", srcIPv6err,
		)
	}

	sourcePort, err := ipfixMap.GetInt("sourceTransportPort")
	if err != nil {
		return errors.Wrap(err, "input map must contain an int value for 'netflow.sourceTransportPort'")
	}

	destIPv4, destIPv4err := ipfixMap.GetString("destinationIPv4Address")
	destIPv6, destIPv6err := ipfixMap.GetString("destinationIPv6Address")
	if destIPv4err != nil && destIPv6err != nil {
		return errors.Wrapf(
			destIPv4err, "input map must contain a string value for "+
				"'netflow.destinationIPv4Address' or 'netflow.destinationIPv6Address'\n"+
				"Additional Cause:\n %+v", destIPv6err,
		)
	}

	if destIPv4err == nil {
		postNatDestIPv4, err := ipfixMap.GetString("postNATDestinationIPv4Address")
		if err == nil {
			destIPv4 = postNatDestIPv4
		} else if errors.Cause(err) == safemap.ErrTypeMismatch {
			return errors.Wrap(err, "input map contains a non-string value for 'netflow.postNATDestinationIPv4Address'")
		}
	} else if destIPv6err == nil {
		postNatDestIPv6, err := ipfixMap.GetString("postNATDestinationIPv6Address")
		if err == nil {
			destIPv6 = postNatDestIPv6
		} else if errors.Cause(err) == safemap.ErrTypeMismatch {
			return errors.Wrap(err, "input map contains a non-string value for 'netflow.postNATDestinationIPv6Address'")
		}
	}

	destPort, err := ipfixMap.GetInt("destinationTransportPort")
	if err != nil {
		return errors.Wrap(err, "input map must contain an int value for 'netflow.sourceTransportPort'")
	}

	postNATDestPort, err := ipfixMap.GetInt("postNAPTDestinationTransportPort")
	if err == nil {
		destPort = postNATDestPort
	} else if errors.Cause(err) == safemap.ErrTypeMismatch {
		return errors.Wrap(err, "input map contains a non-string value for 'netflow.postNAPTDestinationTransportPort'")
	}

	var flowStart, flowEnd string // RFC3339Nano timestamps

	//Get the Start, and the end times of the flow
	flowStartMillis, flowStartMillisErr := ipfixMap.GetString("flowStartMilliseconds")
	flowEndMillis, flowEndMillisErr := ipfixMap.GetString("flowEndMilliseconds")

	//Also attempt to get the start and end times relative to system init
	flowStartUptimeMillis, flowStartUptimeMillisErr := ipfixMap.GetIntAsInt64("flowStartSysUpTime")
	flowEndUptimeMillis, flowEndUptimeMillisErr := ipfixMap.GetIntAsInt64("flowEndSysUpTime")

	//get the system init time if possible
	systemInitTimeMillis, systemInitTimeMillisecondsOk := f.ipfixExporterAbsUptimes[host]
	//If the system init time isn't present or stored use a relative uptime approach
	systemRelativeMillis, systemRelativeOk := f.ipfixExporterRelUptimes[host]

	for _, timeTypeErr := range []error{flowStartMillisErr, flowEndMillisErr, flowStartUptimeMillisErr, flowEndMillisErr} {
		if errors.Cause(timeTypeErr) == safemap.ErrTypeMismatch {
			return err
		}
	}

	if flowStartMillisErr == nil && flowEndMillisErr == nil {
		//Case 1: We have an absolute start and end time (this is ideal)
		flowStart = flowStartMillis
		flowEnd = flowEndMillis
	} else if flowStartUptimeMillisErr == nil && flowEndUptimeMillisErr == nil && systemInitTimeMillisecondsOk {
		//Case 2: We have an start and end time in milliseconds from system init and
		//  we have the absolute system init time, we can find the absolute start
		//  and end time of each flow, less ideal because of computation time
		flowStartUnixMillis := systemInitTimeMillis + flowStartUptimeMillis
		flowStartUnixSeconds := flowStartUnixMillis / 1000
		flowStartUnixNanos := (flowStartUnixMillis % 1000) * 1000000

		flowEndUnixMillis := systemInitTimeMillis + flowEndUptimeMillis
		flowEndUnixSeconds := flowEndUnixMillis / 1000
		flowEndUnixNanos := (flowEndUnixMillis % 1000) * 1000000

		flowStart = time.Unix(flowStartUnixSeconds, flowStartUnixNanos).Format(time.RFC3339Nano)
		flowEnd = time.Unix(flowEndUnixSeconds, flowEndUnixNanos).Format(time.RFC3339Nano)
	} else if flowStartUptimeMillisErr == nil && flowEndUptimeMillisErr == nil && systemRelativeOk {
		//Case 3: We have an start and end time in milliseconds from system init and
		//  we have a timestamp from when the first flow was made available, less
		//  ideal yet as the time of flow will be off but the beaconing algorithm
		//  still works
		firstFlowTime := systemRelativeMillis.firstFlowTime
		firstFlowMillis := systemRelativeMillis.firstFlowMills

		//By finding the difference in time from the first flow and the new flow
		//  start and end we can calculate the time since the first flow and add that
		//  difference to the time of the first flow
		//Note: since the time.Add function takes a time.Duration (which is a int64
		//  nanoseconds count) we need to convert time to nanoseconds from ms
		//  this is done by miltiplying by 1000000
		flowStartOffsetNanos := (flowStartUptimeMillis - firstFlowMillis) * 1000000
		flowEndOffsetNanos := (flowEndUptimeMillis - firstFlowMillis) * 1000000

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

	octetTotal, totalErr := ipfixMap.GetIntAsInt64("octetTotalCount")
	if totalErr != nil {
		if errors.Cause(totalErr) == safemap.ErrTypeMismatch {
			return errors.Wrap(totalErr, "input map contains non-int value for 'netflow.octetTotalCount'")
		}
		//delta counts CAN be total counts by RFC definition >.<"
		octetDelta, deltaErr := ipfixMap.GetIntAsInt64("octetDeltaCount")
		if deltaErr != nil {
			return errors.Wrapf(
				totalErr, "input map must contain an int value for "+
					"'netflow.octetTotalCount' or 'netflow.octetDeltaCount'\n"+
					"Additional Cause:\n %+v", deltaErr,
			)
		}
		octetTotal = octetDelta
	}

	packetTotal, totalErr := ipfixMap.GetIntAsInt64("packetTotalCount")
	if totalErr != nil {
		if errors.Cause(totalErr) == safemap.ErrTypeMismatch {
			return errors.Wrap(totalErr, "input map contains non-int value for 'netflow.packetTotalCount'")
		}
		//delta counts CAN be total counts by RFC definition >.<"
		packetDelta, deltaErr := ipfixMap.GetIntAsInt64("packetDeltaCount")
		if deltaErr != nil {
			return errors.Wrapf(
				totalErr, "input map must contain an int value for "+
					"'netflow.packetTotalCount' or 'netflow.packetDeltaCount'\n"+
					"Additional Cause:\n %+v", deltaErr,
			)
		}
		packetTotal = packetDelta
	}

	protocolID, err := ipfixMap.GetInt("protocolIdentifier")
	if err != nil {
		return errors.Wrap(err, "input map must contain int value for 'netflow.protocolIdentifier'")
	}

	//assume EndOfFlow if flowEndReason is not present
	flowEndReason := input.EndOfFlow
	flowEndReasonInt, err := ipfixMap.GetInt("flowEndReason")
	if err == nil {
		flowEndReason = input.FlowEndReason(flowEndReasonInt)
	} else if errors.Cause(err) == safemap.ErrTypeMismatch {
		return errors.Wrap(err, "input map contains non-int value for 'netflow.flowEndReason'")
	}

	//Fill in the flow now that we know we have all the data
	if srcIPv4err != nil {
		outputFlow.Netflow.SourceIPv4 = sourceIPv4
	}
	if srcIPv6err != nil {
		outputFlow.Netflow.SourceIPv6 = sourceIPv6
	}

	outputFlow.Netflow.SourcePort = uint16(sourcePort)

	if destIPv4err != nil {
		outputFlow.Netflow.DestinationIPv4 = destIPv4
	}
	if destIPv6err != nil {
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
func (f *FlowDeserializer) fillFromNetflowv9BSONMap(netflowMap safemap.SafeMap, outputFlow *Flow) error {
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
	octetTotal, err := iFaceToInt64(octetTotalIface)
	if err != nil {
		return err
	}

	packetTotalIface, ok := netflowMap["in_pkts"]
	if !ok {
		return errors.New("input map must contain key 'netflow.in_pkts'")
	}
	packetTotal, err := iFaceToInt64(packetTotalIface)
	if err != nil {
		return err
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
func (f *FlowDeserializer) fillFromNetflowv5BSONMap(netflowMap safemap.SafeMap, outputFlow *Flow) error {
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
	octetTotal, err := iFaceToInt64(octetTotalIface)
	if err != nil {
		return err
	}

	packetTotalIface, ok := netflowMap["in_pkts"]
	if !ok {
		return errors.New("input map must contain key 'netflow.in_pkts'")
	}
	packetTotal, err := iFaceToInt64(packetTotalIface)
	if err != nil {
		return err
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

//DeserializeNextMap reads the data from a bson map and inserts
//it into the output flow, returning nil if the conversion was successful.
//This method is used for filtering input data and adapting
//multiple versions of netflow records to the same data type.
//If the inputMap contains data that must be maintained as state,
//for example, IPFIX's systemInitTimeMilliseconds, the state will be retained
//even if the flow is only partially filled and an error is returned.
func (f *FlowDeserializer) DeserializeNextMap(inputMap safemap.SafeMap, outputFlow *Flow) error {
	id, err := inputMap.GetObjectID("_id")
	if err != nil {
		return errors.Wrap(err, "input map must contain an ObjectId value for '_id'")
	}
	host, err := inputMap.GetString("host")
	if err != nil {
		return errors.Wrap(err, "input map must contain a string value for 'host'")
	}
	netflow, err := inputMap.GetSafeMap("netflow")
	if err != nil {
		return errors.Wrap(err, "input map must contain a map value for 'netflow'")
	}
	version, err := netflow.GetInt("version")
	if err != nil {
		return errors.Wrap(err, "input map must contain an int value for 'netflow.version'")
	}

	//set the loaded contents
	outputFlow.ID = id
	outputFlow.Host = host
	outputFlow.Netflow.Version = uint8(version)

	//Version must be 10 or 9 or 5
	if outputFlow.Netflow.Version == 10 {
		//handle recording systemInitTimeMilliseconds
		f.updateExporterAbsUptimes(netflowMap, host)
		//theres a chance that systemInitTimeMilliseconds
		//came inside a flow record, parse the rest out just in case...
		//unfortunately, we can't tell option records from flow records
		f.updateExporterRelUptimes(netflowMap, host)

		return f.fillFromIPFIXBSONMap(netflowMap, outputFlow, host)
	} else if outputFlow.Netflow.Version == 9 {
		return f.fillFromNetflowv9BSONMap(netflowMap, outputFlow)
	} else if outputFlow.Netflow.Version == 5 {
		return f.fillFromNetflowv5BSONMap(netflowMap, outputFlow)
	}
	return errors.Errorf("unsupported netflow version: %d", outputFlow.Netflow.Version)
}