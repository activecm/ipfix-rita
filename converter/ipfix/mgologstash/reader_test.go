package mgologstash_test

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/ipfix/mgologstash"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	env := integrationtest.GetDependencies(t).GetFreshEnvironment(t)

	buff := mgologstash.NewIDBuffer(env.DB.NewInputConnection(), env.Logger)
	reader := mgologstash.NewReader(buff, 2*time.Second, env.Logger)

	c := env.DB.NewInputConnection()
	err := c.Insert(getTestFlow1())
	require.Nil(t, err)
	err = c.Insert(getTestFlow2())
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	flows, errs := reader.Drain(ctx)

	type testResult struct {
		testPass bool
		testDesc string
		testData interface{}
	}

	flowTestResults := make(chan testResult, 50)
	errorTestResults := make(chan testResult, 50)
	wg := sync.WaitGroup{}
	go func(flowTestResults chan<- testResult, flows <-chan ipfix.Flow, wg *sync.WaitGroup) {
		f, ok := <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok := f.(*mgologstash.Flow)
		flowTestResults <- testResult{ok, "flow is *mgologstash.Flow", nil}

		flow1 := getTestFlow1()
		flow1.ID = outFlow.ID
		flowTestResults <- testResult{reflect.DeepEqual(flow1, outFlow), "flow1 read correctly", []interface{}{outFlow, flow1}}

		f, ok = <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok = f.(*mgologstash.Flow)
		flowTestResults <- testResult{ok, "flow is *mgologstash.Flow", nil}

		flow2 := getTestFlow2()
		flow2.ID = outFlow.ID
		flowTestResults <- testResult{reflect.DeepEqual(flow2, outFlow), "flow2 read correctly", []interface{}{outFlow, flow2}}

		f, ok = <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok = f.(*mgologstash.Flow)
		flowTestResults <- testResult{ok, "flow is *mgologstash.Flow", nil}

		flow3 := getTestFlow3()
		flow3.ID = outFlow.ID
		flowTestResults <- testResult{reflect.DeepEqual(flow3, outFlow), "flow3 read correctly", []interface{}{outFlow, flow3}}
		close(flowTestResults)
	}(flowTestResults, flows, &wg)

	go func(errorTestResults chan<- testResult, errs <-chan error, wg *sync.WaitGroup) {
		e, ok := <-errs
		errorTestResults <- testResult{!ok, "no errors should be recieved", e}
		close(errorTestResults)
	}(errorTestResults, errs, &wg)

	time.Sleep(2 * time.Second)
	err = c.Insert(getTestFlow3())
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
	cancel()

	for result := range flowTestResults {
		if !result.testPass {
			msg := fmt.Sprintf("FAIL: %s\n", result.testDesc)
			if result.testData != nil {
				msg = fmt.Sprintf("%sData: %s\n", msg, spew.Sdump(result.testData))
			}
			t.Fatal(msg)
		}
	}

	for result := range errorTestResults {
		if !result.testPass {
			msg := fmt.Sprintf("FAIL: %s\n", result.testDesc)
			if result.testData != nil {
				msg = fmt.Sprintf("%s\tData: %+v\n", msg, result.testData)
			}
			t.Fatal(msg)
		}
	}

	count, err := c.Count()
	require.Nil(t, err)
	require.Equal(t, 0, count)
}
