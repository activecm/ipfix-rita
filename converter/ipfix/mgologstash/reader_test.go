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
	err := c.Insert(mgologstash.Flow{
		Host: "A",
	})
	require.Nil(t, err)
	err = c.Insert(mgologstash.Flow{
		Host: "B",
	})
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	flows, errors := reader.Drain(ctx)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(t *testing.T, flows <-chan ipfix.Flow, wg *sync.WaitGroup) {
		f := <-flows
		require.Equal(t, "A", f.Exporter())
		t.Log("Read 1st record")
		f = <-flows
		require.Equal(t, "B", f.Exporter())
		t.Log("Read 2nd record")
		f = <-flows
		require.Equal(t, "C", f.Exporter())
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
	err = c.Insert(mgologstash.Flow{
		Host: "C",
	})
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
