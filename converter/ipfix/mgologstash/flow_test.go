package mgologstash_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/ipfix/mgologstash"
	"github.com/stretchr/testify/require"
)

func TestParseLogstashTime(t *testing.T) {
	flow := mgologstash.Flow{}
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flowStart, err := flow.FlowStartMilliseconds()
	require.Nil(t, err)
	flowEnd, err := flow.FlowEndMilliseconds()
	require.Nil(t, err)
	require.Equal(t, uint64(1525473400*1000)+766, flowStart)
	require.Equal(t, uint64(1525473400*1000)+960, flowEnd)
}

func TestV4V6Address(t *testing.T) {
	flow := mgologstash.Flow{}
	flow.Netflow.SourceIPv4 = "A"
	flow.Netflow.DestinationIPv4 = "B"
	require.Equal(t, flow.SourceIPAddress(), "A")
	require.Equal(t, flow.DestinationIPAddress(), "B")
	flow = mgologstash.Flow{}
	flow.Netflow.SourceIPv6 = "C"
	flow.Netflow.DestinationIPv6 = "D"
	require.Equal(t, flow.SourceIPAddress(), "C")
	require.Equal(t, flow.DestinationIPAddress(), "D")
}

func TestInheritance(t *testing.T) {
	var flow interface{} = &mgologstash.Flow{}
	_, ok := flow.(ipfix.Flow)
	require.True(t, ok)
}
