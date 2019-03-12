package filter_test

import (
	"net"
	"testing"

	"github.com/activecm/ipfix-rita/converter/filter"
	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/stretchr/testify/require"
)

func TestFlowBlacklist(t *testing.T) {
	internalNetStrs := []string{"8.8.8.8/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	neverIncludeNetStrs := []string{"8.8.8.8/32", "4.4.4.4/32"}
	alwaysIncludeNetStrs := []string{"8.8.8.8/32"}

	internalNets := make([]net.IPNet, len(internalNetStrs))
	neverIncludeNets := make([]net.IPNet, len(neverIncludeNetStrs))
	alwaysIncludeNets := make([]net.IPNet, len(alwaysIncludeNetStrs))

	for _, netStr := range internalNetStrs {
		_, net, err := net.ParseCIDR(netStr)
		require.Nil(t, err)
		internalNets = append(internalNets, *net)
	}

	for _, netStr := range neverIncludeNetStrs {
		_, net, err := net.ParseCIDR(netStr)
		require.Nil(t, err)
		neverIncludeNets = append(neverIncludeNets, *net)
	}

	for _, netStr := range alwaysIncludeNetStrs {
		_, net, err := net.ParseCIDR(netStr)
		require.Nil(t, err)
		alwaysIncludeNets = append(alwaysIncludeNets, *net)
	}

	flowBlackList := filter.NewFlowBlacklist(
		internalNets,
		neverIncludeNets,
		alwaysIncludeNets,
	)

	type flowBlacklistTestCase struct {
		src string
		dst string
		out bool
	}

	testCases := []flowBlacklistTestCase{
		flowBlacklistTestCase{ // internal and external
			src: "10.10.10.10",
			dst: "34.10.10.11",
			out: false,
		},
		flowBlacklistTestCase{ // internal and internal
			src: "10.10.10.10",
			dst: "10.10.10.11",
			out: true,
		},
		flowBlacklistTestCase{ // internal and internal
			src: "192.168.1.1",
			dst: "192.168.1.1",
			out: true,
		},
		flowBlacklistTestCase{ // internal and internal
			src: "192.168.1.1",
			dst: "192.168.2.1",
			out: true,
		},
		flowBlacklistTestCase{ // internal and always include
			src: "8.8.8.8",
			dst: "192.168.2.1",
			out: false,
		},
		flowBlacklistTestCase{ // internal and always include
			src: "192.168.2.1",
			dst: "8.8.8.8",
			out: false,
		},
		flowBlacklistTestCase{ //src and dst on opposing lists
			src: "8.8.8.8",
			dst: "8.8.4.4",
			out: false,
		},
		flowBlacklistTestCase{ //src and dst on opposing lists
			src: "8.8.4.4",
			dst: "8.8.8.8",
			out: false,
		},
		flowBlacklistTestCase{ // external and external
			src: "24.10.10.10",
			dst: "34.10.10.11",
			out: true,
		},
		flowBlacklistTestCase{ // external and external
			src: "139.130.4.5",
			dst: "208.67.222.222",
			out: true,
		},
		flowBlacklistTestCase{ // internal and never include
			src: "10.10.10.10",
			dst: "4.4.4.4",
			out: true,
		},
	}

	for _, testCase := range testCases {
		flow := input.FlowMock{
			MockSourceIPAddress:      testCase.src,
			MockDestinationIPAddress: testCase.dst,
		}
		match, err := flowBlackList.Match(&flow)
		require.Nil(t, err)
		require.Equal(t, testCase.out, match)
	}

	//src unparseable
	flow := input.FlowMock{
		MockSourceIPAddress:      "nonsense",
		MockDestinationIPAddress: "34.10.10.11",
	}
	_, err := flowBlackList.Match(&flow)
	require.NotNil(t, err)

	//dst unparseable
	flow = input.FlowMock{
		MockSourceIPAddress:      "34.10.10.11",
		MockDestinationIPAddress: "nonsense",
	}
	_, err = flowBlackList.Match(&flow)
	require.NotNil(t, err)

}
