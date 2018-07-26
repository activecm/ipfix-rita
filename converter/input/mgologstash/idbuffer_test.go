package mgologstash_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/input/mgologstash"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/stretchr/testify/require"
)

func TestIDBulkBuffer(t *testing.T) {
	env := integrationtest.GetDependencies(t).Env
	buffer := mgologstash.NewIDBulkBuffer(env.DB.NewInputConnection(), 1000, env.Logger)
	testBufferOrder(buffer, env, t)
}

func testBufferOrder(buffer mgologstash.Buffer, env environment.Environment, t *testing.T) {
	testFlow1 := getTestFlow1()
	testFlow2 := getTestFlow2()

	c := env.DB.NewInputConnection()
	err := c.Insert(testFlow1)
	require.Nil(t, err)
	err = c.Insert(testFlow2)
	require.Nil(t, err)
	count, err := c.Count()
	require.Nil(t, err)
	require.Equal(t, 2, count)
	c.Database.Session.Close()

	var flow mgologstash.Flow

	more := buffer.Next(&flow)
	require.True(t, more)
	require.Nil(t, buffer.Err())

	testFlow1.ID = flow.ID
	require.Equal(t, testFlow1, &flow)

	more = buffer.Next(&flow)
	require.True(t, more)
	require.Nil(t, buffer.Err())

	testFlow2.ID = flow.ID
	require.Equal(t, testFlow2, &flow)

	//flow must remain unchanged
	more = buffer.Next(&flow)
	require.False(t, more)
	require.Nil(t, buffer.Err())

	require.Equal(t, testFlow2, &flow)
	buffer.Close()
}

//TODO: TestSkipInvalidFlows with various input objects (ie valid IPFIX records with different fields)
