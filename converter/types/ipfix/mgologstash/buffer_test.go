package mgologstash_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/environmenttest"
	"github.com/activecm/ipfix-rita/converter/types/ipfix/mgologstash"
	"github.com/stretchr/testify/require"
)

func TestBuffer(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()
	c := env.DB.DB("IPFIX").C("in")
	err := c.Insert(mgologstash.Flow{
		Host: "A",
	})
	require.Nil(t, err)
	err = c.Insert(mgologstash.Flow{
		Host: "B",
	})
	require.Nil(t, err)
	count, err := c.Count()
	require.Nil(t, err)
	require.Equal(t, 2, count)

	buffer := mgologstash.NewBuffer(c)
	var flow mgologstash.Flow
	more := buffer.Next(&flow)
	require.Equal(t, "A", flow.Exporter())
	require.True(t, more)
	more = buffer.Next(&flow)
	require.Equal(t, "B", flow.Exporter())
	require.True(t, more)
	more = buffer.Next(&flow)
	//flow must remain unchanged
	require.Equal(t, "B", flow.Exporter())
	require.False(t, more)
	require.Nil(t, buffer.Err())
	buffer.Close()

}
