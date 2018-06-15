package protocols_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/stretchr/testify/require"
)

func TestProtocols(t *testing.T) {
	require.Equal(t, protocols.Identifier(1), protocols.ICMP)
	require.Equal(t, protocols.Identifier(6), protocols.TCP)
	require.Equal(t, protocols.Identifier(17), protocols.UDP)
	require.Equal(t, protocols.Identifier(58), protocols.IPv6_ICMP)
	require.Equal(t, protocols.Identifier(132), protocols.SCTP)
	require.Equal(t, protocols.Identifier(142), protocols.ROHC)
}
