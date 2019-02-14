package data_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/input/logstash/data"
	"github.com/stretchr/testify/require"
)

func TestParseLogstashTime(t *testing.T) {
	flow := data.Flow{}
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flowStart, err := flow.FlowStartMilliseconds()
	require.Nil(t, err)
	flowEnd, err := flow.FlowEndMilliseconds()
	require.Nil(t, err)
	require.Equal(t, int64(1525473400*1000)+766, flowStart)
	require.Equal(t, int64(1525473400*1000)+960, flowEnd)
}

func TestV4Address(t *testing.T) {
	flow := data.Flow{}
	flow.Netflow.SourceIPv4 = "A"
	flow.Netflow.DestinationIPv4 = "B"
	require.Equal(t, flow.SourceIPAddress(), "A")
	require.Equal(t, flow.DestinationIPAddress(), "B")
}

func TestV6Address(t *testing.T) {
	flow := data.Flow{}
	flow.Netflow.SourceIPv6 = "C"
	flow.Netflow.DestinationIPv6 = "D"
	require.Equal(t, flow.SourceIPAddress(), "C")
	require.Equal(t, flow.DestinationIPAddress(), "D")
}

func TestInheritance(t *testing.T) {
	var flow interface{} = &data.Flow{}
	_, ok := flow.(input.Flow)
	require.True(t, ok)
}
