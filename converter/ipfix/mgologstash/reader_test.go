package mgologstash_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/activecm/ipfix-rita/converter/environmenttest"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/ipfix/mgologstash"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()

	buff := mgologstash.NewIDBuffer(env.DB.NewInputConnection())
	reader := mgologstash.NewReader(buff, 2*time.Second)

	c := env.DB.NewInputConnection()
	err := c.Insert(getTestFlow1())
	require.Nil(t, err)
	err = c.Insert(getTestFlow2())
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	flows, errors := reader.Drain(ctx)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(t *testing.T, flows <-chan ipfix.Flow, wg *sync.WaitGroup) {
		f, ok := <-flows
		require.True(t, ok)
		outFlow, ok := f.(*mgologstash.Flow)
		require.True(t, ok)

		flow1 := getTestFlow1()
		flow1.ID = outFlow.ID
		require.Equal(t, flow1, outFlow)
		t.Log("Read 1st record")

		f, ok = <-flows
		require.True(t, ok)
		outFlow, ok = f.(*mgologstash.Flow)
		require.True(t, ok)

		flow2 := getTestFlow2()
		flow2.ID = outFlow.ID
		require.Equal(t, flow2, outFlow)
		t.Log("Read 2nd record")

		f, ok = <-flows
		require.True(t, ok)
		outFlow, ok = f.(*mgologstash.Flow)
		require.True(t, ok)

		flow3 := getTestFlow3()
		flow3.ID = outFlow.ID
		require.Equal(t, flow3, outFlow)
		t.Log("Read delayed 3rd record")
		wg.Done()
	}(t, flows, &wg)

	wg.Add(1)
	go func(t *testing.T, errors <-chan error, wg *sync.WaitGroup) {
		e := <-errors
		require.Equal(t, e, context.Canceled)
		t.Log("Read context cancelled")
		wg.Done()
	}(t, errors, &wg)

	time.Sleep(2 * time.Second)
	err = c.Insert(getTestFlow3())
	require.Nil(t, err)
	t.Log("Wrote three records")
	time.Sleep(2 * time.Second)
	cancel()
	t.Log("Cancelled reader context")
	wg.Wait()
	count, err := c.Count()
	require.Nil(t, err)
	require.Equal(t, 0, count)
}
