package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//FlowEndReason Represents IPFIX Information Export #136
type FlowEndReason int32

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
)

//ProtocolNumberMap represents the IANA Protocol Numbers
//https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
var ProtocolNumberMap = map[int]string{
	1:  "ICMP",
	6:  "TCP",
	17: "UDP",
	58: "IPv6-ICMP",
}

//LogstashIPFIX represents an IPFIX record stored in MongoDB via Logstash
type LogstashIPFIX struct {
	ID      bson.ObjectId `bson:"_id"`
	Host    string        `bson:"host"` //Host is the metering process host
	Netflow struct {
		SourceIPv4 string `bson:"sourceIPv4Address,omitempty"`
		SourceIPv6 string `bson:"sourceIPv6Address,omitempty"`
		SourcePort int32  `bson:"sourceTransportPort"`

		DestinationIPv4 string `bson:"destinationIPv4Address,omitempty"`
		DestinationIPv6 string `bson:"destinationIPv6Address,omitempty"`
		DestinationPort int32  `bson:"destinationTransportPort"`

		// NOTE: We may need fields for other time units
		FlowStartTime string `bson:"flowStartMilliseconds"`
		FlowEndTime   string `bson:"flowEndMilliseconds"`

		OrigIPBytes int32 `bson:"octetTotalCount"`
		OrigPkts    int32 `bson:"packetTotalCount"`

		ProtocolID       int32         `bson:"protocolIdentifier"`
		IPClassOfService int32         `bson:"ipClassOfService"`
		VlanID           int32         `bson:"vlanID"`
		FlowEndReason    FlowEndReason `bson:"flowEndReason"`
		IPFIXVersion     int32         `bson:"version"`
	} `bson:"netflow"`
}

//GetFlowKeyView fills in a LogstashIPFIXFlowKeyView with the
//corresponding Flow Key for this IPFIX record
func (l *LogstashIPFIX) GetFlowKeyView(out *LogstashIPFIXFlowKeyView) {
	if l.Netflow.DestinationIPv4 != "" {
		out.Netflow.DestinationIPv4 = &l.Netflow.DestinationIPv4
	}
	if l.Netflow.DestinationIPv6 != "" {
		out.Netflow.DestinationIPv6 = &l.Netflow.DestinationIPv6
	}
	out.Netflow.DestinationPort = &l.Netflow.DestinationPort

	if l.Netflow.SourceIPv4 != "" {
		out.Netflow.SourceIPv4 = &l.Netflow.SourceIPv4
	}
	if l.Netflow.SourceIPv6 != "" {
		out.Netflow.SourceIPv6 = &l.Netflow.SourceIPv6
	}
	out.Netflow.SourcePort = &l.Netflow.SourcePort

	out.Netflow.ProtocolID = &l.Netflow.ProtocolID
}

//LogstashIPFIXFlowKeyView provides a view of an IPFIX record
//detailing the Flow Key fields
type LogstashIPFIXFlowKeyView struct {
	Netflow struct {
		SourceIPv4 *string
		SourceIPv6 *string
		SourcePort *int32

		DestinationIPv4 *string
		DestinationIPv6 *string
		DestinationPort *int32
		ProtocolID      *int32
	}
}

//ToMgoQueryObj converts the information referenced in the view
//to a bson Map which is useful for finding Logstash IPFIX records
//which match this Flow Key
func (fk *LogstashIPFIXFlowKeyView) ToMgoQueryObj() bson.M {
	mgoQueryObj := make(bson.M)
	if fk.Netflow.SourceIPv4 != nil {
		mgoQueryObj["netflow.sourceIPv4Address"] = *fk.Netflow.SourceIPv4
	}
	if fk.Netflow.SourceIPv6 != nil {
		mgoQueryObj["netflow.sourceIPv6Address"] = *fk.Netflow.SourceIPv6
	}
	mgoQueryObj["netflow.sourceTransportPort"] = *fk.Netflow.SourcePort

	if fk.Netflow.DestinationIPv4 != nil {
		mgoQueryObj["netflow.destinationIPv4Address"] = *fk.Netflow.DestinationIPv4
	}
	if fk.Netflow.DestinationIPv6 != nil {
		mgoQueryObj["netflow.destinationIPv6Address"] = *fk.Netflow.DestinationIPv6
	}
	mgoQueryObj["netflow.destinationTransportPort"] = *fk.Netflow.DestinationPort
	return mgoQueryObj
}

//Flip swaps the fields referenced in this Flow Key view which is
//useful for finding IPFIX records representing the corresponding flows
//which go from the destination host to the source host.
func (fk *LogstashIPFIXFlowKeyView) Flip() {
	var temp *string
	temp = fk.Netflow.DestinationIPv4
	fk.Netflow.DestinationIPv4 = fk.Netflow.SourceIPv4
	fk.Netflow.SourceIPv4 = temp

	temp = fk.Netflow.DestinationIPv6
	fk.Netflow.DestinationIPv6 = fk.Netflow.SourceIPv6
	fk.Netflow.SourceIPv6 = temp

	var tempInt *int32
	tempInt = fk.Netflow.DestinationPort
	fk.Netflow.DestinationPort = fk.Netflow.SourcePort
	fk.Netflow.SourcePort = tempInt
}

//LogstashIPFIXQueryAggregate records the LogstashIPFIX record which was
//used to produce a query as well as the results of that query
type LogstashIPFIXQueryAggregate struct {
	Query       LogstashIPFIX   `bson:"query"`
	Records     []LogstashIPFIX `bson:"records"`
	RecordCount int             `bson:"recordCount"`
}

//SameKeyQueriesColl is the name of the collection which holds same-flow-key queries
const SameKeyQueriesColl string = "same-flowkey-queries"

//FlipKeyQueriesColl is the name of the collection which holds flip-flow-key queries
const FlipKeyQueriesColl string = "flip-flowkey-queries"

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s <MongoDB URI> <Logstash IPFIX DB> <Logstash IPFIX Collection> [Expiration Timeout Minutes]\n", os.Args[0])
		os.Exit(-1)
	}
	uri, sourceDB, sourceCollection := os.Args[1], os.Args[2], os.Args[3]
	var expirationTimeout time.Duration //zero is a flag value for no timeout
	if len(os.Args) > 4 {
		expirationMinutes, err := strconv.Atoi(os.Args[4])
		if err != nil {
			panic(err)
		}
		expirationTimeout = time.Duration(expirationMinutes) * time.Minute
	}
	ssn, err := mgo.Dial(uri)
	if err != nil {
		panic(err)
	}
	dbExists, collExists := doesCollectionExist(ssn, sourceDB, sourceCollection)
	if !dbExists || !collExists {
		fmt.Printf("Ensure the source IPFIX database is correctly loaded at:")
		fmt.Printf("DB: %s\nCollection: %s\n", sourceDB, sourceCollection)
		panic(errors.New("source collection does not exist"))
	}
	ensureLogstashIPFIXIndexes(ssn, sourceDB, sourceCollection)
	deleteOldResults(ssn, sourceDB)
	sortIPFIXByFlowKeyCounts(ssn, sourceDB, sourceCollection, expirationTimeout)
	report(ssn, sourceDB, expirationTimeout)
}

func doesCollectionExist(ssn *mgo.Session, sourceDB, sourceColl string) (bool, bool) {
	dbNames, err := ssn.DatabaseNames()
	if err != nil {
		panic(err)
	}
	found := false
	for i := range dbNames {
		if dbNames[i] == sourceDB {
			found = true
			break
		}
	}
	if !found {
		return false, false
	}

	collNames, err := ssn.DB(sourceDB).CollectionNames()
	if err != nil {
		panic(err)
	}
	found = false
	for i := range collNames {
		if collNames[i] == sourceColl {
			found = true
			break
		}
	}
	return true, found
}

func deleteOldResults(ssn *mgo.Session, sourceDB string) {
	logstashDB := ssn.DB(sourceDB)
	colls, err := logstashDB.CollectionNames()
	if err != nil {
		panic(err)
	}
	for i := range colls {
		if colls[i] == SameKeyQueriesColl ||
			colls[i] == FlipKeyQueriesColl {
			logstashDB.C(colls[i]).DropCollection()
		}
	}
}

func ensureLogstashIPFIXIndexes(ssn *mgo.Session, sourceDB, sourceColl string) {
	coll := ssn.DB(sourceDB).C(sourceColl)
	ipv4 := mgo.Index{
		Name: "5 Tuple IPv4",
		Key: []string{
			"netflow.destinationIPv4Address",
			"netflow.destinationTransportPort",
			"netflow.sourceIPv4Address",
			"netflow.sourceTransportPort",
			"netflow.protocolIdentifier",
		},
	}

	ipv6 := mgo.Index{
		Name: "5 Tuple IPv6",
		Key: []string{
			"netflow.destinationIPv6Address",
			"netflow.destinationTransportPort",
			"netflow.sourceIPv6Address",
			"netflow.sourceTransportPort",
			"netflow.protocolIdentifier",
		},
	}
	err := coll.EnsureIndex(ipv4)
	if err != nil && !strings.HasPrefix(err.Error(), "Index must have unique name") {
		panic(err)
	}

	err = coll.EnsureIndex(ipv6)
	if err != nil && !strings.HasPrefix(err.Error(), "Index must have unique name") {
		panic(err)
	}
}

func sortIPFIXByFlowKeyCounts(ssn *mgo.Session, sourceDB, sourceColl string, expirationThreshold time.Duration) {
	//Iterate over the IPFIX records
	logstashDB := ssn.DB(sourceDB)
	logstashIPFIXcoll := logstashDB.C(sourceColl)
	iter := logstashIPFIXcoll.Find(nil).Iter()
	var buffObj LogstashIPFIX
	for iter.Next(&buffObj) {
		//Calculate the finish time for this record to simulate expiration
		buffObjFinishTime, err := time.Parse(time.RFC3339, buffObj.Netflow.FlowEndTime)
		if err != nil {
			panic(err)
		}

		//Get the Mongo Query object for the Flow Key matching this record
		var buffObjFlowKeyView LogstashIPFIXFlowKeyView
		buffObj.GetFlowKeyView(&buffObjFlowKeyView)
		sameQueryObj := buffObjFlowKeyView.ToMgoQueryObj()

		//Iterate over and filter the IPFIX records matching this record's Flow Key
		sameQuery := logstashIPFIXcoll.Find(sameQueryObj)
		sameIter := sameQuery.Iter()
		var sameBuffer []LogstashIPFIX
		var sameBuffObj LogstashIPFIX
		for sameIter.Next(&sameBuffObj) {
			//For the same Flow Key lookup, we don't want the original record
			if sameBuffObj.ID == buffObj.ID {
				continue
			}
			//If a threshold was set, only select records which wouldn't have
			//expired yet
			sameFinishTime, err := time.Parse(time.RFC3339, sameBuffObj.Netflow.FlowEndTime)
			if err != nil {
				panic(err)
			}

			if expirationThreshold != 0 {
				//Check the time between flows to see if the record would have expired
				var timeBetweenFlows time.Duration
				if sameFinishTime.After(buffObjFinishTime) {
					timeBetweenFlows = sameFinishTime.Sub(buffObjFinishTime)
				} else {
					timeBetweenFlows = buffObjFinishTime.Sub(sameFinishTime)
				}

				//We check if the absolute value of the time between flows
				//is greater than the expiration. In a real time situation,
				//we only see the past; however, we would set each new record
				//to the side for some time window and keep updating the
				//record as new connections come in.
				//Here, in batch land, we create N records rather than just 1
				//so we check +/- the expiration time to see if this record
				//would be matched in the future (in addition to checking for
				//matches in the past).
				if timeBetweenFlows > expirationThreshold {
					continue
				}
			}

			//Checks passed, add it to the flow aggregate
			sameBuffer = append(sameBuffer, sameBuffObj)
		}
		if sameIter.Err() != nil {
			panic(sameIter.Err())
		}

		//Insert the aggregate record based on how many records matched the query
		sameAggregate := LogstashIPFIXQueryAggregate{
			Query:       buffObj,
			Records:     sameBuffer,
			RecordCount: len(sameBuffer),
		}

		tgtColl := logstashDB.C(SameKeyQueriesColl)
		tgtColl.Insert(sameAggregate)

		// Next do the same thing for the flipped Flow Key
		// to find the maching flows in the other direction
		buffObjFlowKeyView.Flip()
		flipQueryObj := buffObjFlowKeyView.ToMgoQueryObj()

		//Iterate over and filter the IPFIX entries which may be the
		//corresponding Flows in the other direction
		flipQuery := logstashIPFIXcoll.Find(flipQueryObj)
		flipIter := flipQuery.Iter()
		var flipBuffer []LogstashIPFIX
		var flipBuffObj LogstashIPFIX
		for flipIter.Next(&flipBuffObj) {
			flipFinishTime, err := time.Parse(time.RFC3339, flipBuffObj.Netflow.FlowEndTime)
			if err != nil {
				panic(err)
			}

			if expirationThreshold != 0 {
				//Check the time between flows to see if the record would have expired
				var timeBetweenFlows time.Duration
				if flipFinishTime.After(buffObjFinishTime) {
					timeBetweenFlows = flipFinishTime.Sub(buffObjFinishTime)
				} else {
					timeBetweenFlows = buffObjFinishTime.Sub(flipFinishTime)
				}

				//We check if the absolute value of the time between flows
				//is greater than the expiration. In a real time situation,
				//we only see the past; however, we would set each new record
				//to the side for some time window and keep updating the
				//record as new connections come in.
				//Here, in batch land, we create N records rather than just 1
				//so we check +/- the expiration time to see if this record
				//would be matched in the future (in addition to checking for
				//matches in the past).
				if timeBetweenFlows > expirationThreshold {
					continue
				}
			}
			flipBuffer = append(flipBuffer, flipBuffObj)
		}
		if flipIter.Err() != nil {
			panic(flipIter.Err())
		}

		//Insert the aggregate for the flip query for this IPFIX record
		flipAggregate := LogstashIPFIXQueryAggregate{
			Query:       buffObj,
			Records:     flipBuffer,
			RecordCount: len(flipBuffer),
		}

		tgtColl = logstashDB.C(FlipKeyQueriesColl)
		tgtColl.Insert(flipAggregate)
	}
}

func report(ssn *mgo.Session, sourceDB string, expirationTimeout time.Duration) {
	fmt.Printf("IPFIX Flow Stitching Data Analysis Report\n")
	if expirationTimeout != 0 {
		fmt.Printf("Configured Expiration Timeout: %s\n", expirationTimeout.String())
	} else {
		fmt.Printf("Configured Expiration Timeout: None\n")
	}
	fmt.Printf("Same Flow Key Query Statistics:\n")
	reportQueryAggRecordCounts(ssn, sourceDB, SameKeyQueriesColl)
	fmt.Printf("\nSame Flow Key Queries by Transport Protocol\n")
	fmt.Printf("For Same Flow Key Queries, Easy Queries are Queries Where # of Flow Key Matches = 0\n")
	reportHardQueryAggsByTransport(ssn, sourceDB, SameKeyQueriesColl, 0)
	fmt.Printf("\n\nFlipped Flow Key Query Statistics:\n")
	reportQueryAggRecordCounts(ssn, sourceDB, FlipKeyQueriesColl)
	fmt.Printf("\nFlipped Flow Key Queries by Transport Protocol\n")
	fmt.Printf("For Flipped Flow Key Queries, Easy Queries are Queries Where # of Flow Key Matches = 0 or 1\n")
	reportHardQueryAggsByTransport(ssn, sourceDB, FlipKeyQueriesColl, 1)
}

func reportQueryAggRecordCounts(ssn *mgo.Session, sourceDB, aggColl string) {
	totalNumQueries, err := ssn.DB(sourceDB).C(aggColl).Count()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Total number of queries: %d\n", totalNumQueries)
	fmt.Printf("# of Flow Key Matches, # of Queries, Percent of Queries\n")
	queryRecordCountsIter := ssn.DB(sourceDB).C(aggColl).Pipe(
		[]bson.M{
			{
				"$group": bson.M{
					"_id":        "$recordCount",
					"numQueries": bson.M{"$sum": 1},
				},
			},
			{
				"$sort": bson.M{"_id": 1},
			},
		},
	).Iter()

	var queryRecordCount struct {
		KeyMatches int `bson:"_id"`
		NumQueries int `bson:"numQueries"`
	}
	for queryRecordCountsIter.Next(&queryRecordCount) {
		fmt.Printf("%d, %d, %4.2f%%\n", queryRecordCount.KeyMatches, queryRecordCount.NumQueries, float64(queryRecordCount.NumQueries)/float64(totalNumQueries)*100)
	}
	if queryRecordCountsIter.Err() != nil {
		panic(queryRecordCountsIter.Err())
	}
}

func reportHardQueryAggsByTransport(ssn *mgo.Session, sourceDB, aggColl string, maxEasyNumRecords int) {
	fmt.Printf("Transport Protocol, # of Queries, # of Hard Queries, %% of Hard Queries\n")

	transportHardQueryCountsIter := ssn.DB(sourceDB).C(aggColl).Pipe(
		[]bson.M{
			{"$match": bson.M{"recordCount": bson.M{"$gt": maxEasyNumRecords}}},
			{"$group": bson.M{
				"_id":        "$query.netflow.protocolIdentifier",
				"numQueries": bson.M{"$sum": 1},
			}},
		},
	).Iter()

	var transportHardQueryCount struct {
		ProtoID    int `bson:"_id"`
		NumQueries int `bson:"numQueries"`
	}
	for transportHardQueryCountsIter.Next(&transportHardQueryCount) {
		protoTotalNumQueries, err := ssn.DB(sourceDB).C(aggColl).Find(
			bson.M{
				"query.netflow.protocolIdentifier": transportHardQueryCount.ProtoID,
			},
		).Count()
		if err != nil {
			panic(err)
		}
		protoString, ok := ProtocolNumberMap[transportHardQueryCount.ProtoID]
		if !ok {
			protoString = fmt.Sprintf("%d", transportHardQueryCount.ProtoID)
		}
		fmt.Printf(
			"%s, %d, %d, %4.2f%%\n",
			protoString,
			protoTotalNumQueries,
			transportHardQueryCount.NumQueries,
			float64(transportHardQueryCount.NumQueries)/float64(protoTotalNumQueries)*100,
		)
	}
	if transportHardQueryCountsIter.Err() != nil {
		panic(transportHardQueryCountsIter.Err())
	}
}
