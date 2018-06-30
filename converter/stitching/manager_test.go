package stitching_test

import (
	"testing"
	"context"
	"time"
	"github.com/activecm/ipfix-rita/converter/environmenttest"
	"github.com/activecm/ipfix-rita/converter/stitching"
	"github.com/activecm/ipfix-rita/converter/output"
	input "github.com/activecm/ipfix-rita/converter/ipfix/mgologstash"
	"github.com/stretchr/testify/require"
)

/*
Logstash Flow Sample
{
	"_id" : ObjectId("5b371b96ca8fc40c3c001b24"),
	"@timestamp" : "\"2018-06-30T05:54:57.000Z\"",
	"host" : "172.21.0.1",
	"@version" : "1",
	"netflow" : {
		"flowEndReason" : 1,
		"destinationTransportPort" : 53,
		"sourceTransportPort" : 28752,
		"ipClassOfService" : 0,
		"flowStartMilliseconds" : "2018-05-04T22:36:10.493Z",
		"destinationIPv4Address" : "189.6.48.3",
		"sourceIPv4Address" : "104.131.28.214",
		"flowAttributes" : 0,
		"octetTotalCount" : 79,
		"flowEndMilliseconds" : "2018-05-04T22:36:10.618Z",
		"version" : 10,
		"vlanId" : 0,
		"protocolIdentifier" : 17,
		"packetTotalCount" : 1
	}
}

Protocol with identifier
{
	1 : ICMP
	6 : TCP
	17 :UDP
	58 : IPv6_ICMP
	132 : SCTP
	142 : ROHC
}

*/

func TestSingleIcmpFlow(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	sameSessionThreshold := uint64(1000 * 60 * 60) //milliseconds
	var numStitchers int32 = 5
	stitcherBufferSize := 5
	var writer output.SpewRITAConnWriter

	buff := input.NewIDBuffer(env.DB.NewInputConnection())
	reader := input.NewReader(buff, 2*time.Second)

	c := env.DB.NewInputConnection()

	flow := input.Flow{}
	flow.Host = "A"
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flow.Netflow.ProtocolIdentifier = 1

	err := c.Insert(
		flow,
	)

	require.Nil(t, err)

	ctx,cancel := context.WithCancel(context.Background())
	flows,errors := reader.Drain(ctx)

	stitchingManager := stitching.NewManager(sameSessionThreshold, stitcherBufferSize, numStitchers)
	stitchingErrors := stitchingManager.RunAsync(flows, env.DB, writer)
	require.NotNil(t, stitchingErrors)
	require.NotNil(t,errors)
	require.NotNil(t, flows)
	cancel()
}


func TestTwoIcmpFlow(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	sameSessionThreshold := uint64(1000 * 60 * 60) //milliseconds
	var numStitchers int32 = 5
	stitcherBufferSize := 5
	var writer output.SpewRITAConnWriter

	buff := input.NewIDBuffer(env.DB.NewInputConnection())
	reader := input.NewReader(buff, 2*time.Second)

	c := env.DB.NewInputConnection()

	flow := input.Flow{}
	flow.Host = "A"
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flow.Netflow.ProtocolIdentifier = 1
	flow.Netflow.SourceIPv4 = "192.168.1.1"
	flow.Netflow.SourceIPv6 = "192.168.1.2"

	flow1 := input.Flow{}
	flow1.Host = "B"
	flow1.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow1.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flow1.Netflow.ProtocolIdentifier = 1

	err := c.Insert(
		flow,
	)

	require.Nil(t, err)

	err1 := c.Insert(
		flow1,
	)

	require.Nil(t, err1)
	ctx,cancel := context.WithCancel(context.Background())
	flows,errors := reader.Drain(ctx)

	stitchingManager := stitching.NewManager(sameSessionThreshold, stitcherBufferSize, numStitchers)
	stitchingErrors := stitchingManager.RunAsync(flows, env.DB, writer)

	require.NotNil(t, stitchingErrors)
	require.NotNil(t,errors)
	require.NotNil(t, flows)
	cancel()
}