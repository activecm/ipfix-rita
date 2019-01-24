package mongodb_test

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/input/logstash/data"
	"github.com/activecm/ipfix-rita/converter/input/logstash/mongodb"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func getTestFlow1() *data.Flow {
	flow := &data.Flow{
		Host: "A",
	}
	flow.Netflow.SourceIPv4 = "1.1.1.1"
	flow.Netflow.SourcePort = 24846
	flow.Netflow.DestinationIPv4 = "2.2.2.2"
	flow.Netflow.DestinationPort = 53
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:36:40.766Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:36:40.960Z"
	flow.Netflow.OctetTotalCount = int64(math.MaxUint32 + 1)
	flow.Netflow.PacketTotalCount = int64(math.MaxUint32 + 1)
	flow.Netflow.ProtocolIdentifier = protocols.UDP
	flow.Netflow.FlowEndReason = input.ActiveTimeout
	flow.Netflow.Version = 10
	return flow
}

func getTestFlow2() *data.Flow {
	flow := &data.Flow{
		Host: "B",
	}
	flow.Netflow.SourceIPv4 = "2.2.2.2"
	flow.Netflow.SourcePort = 53
	flow.Netflow.DestinationIPv4 = "1.1.1.1"
	flow.Netflow.DestinationPort = 24846
	flow.Netflow.FlowStartMilliseconds = "2018-05-04T22:40:40.555Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-04T22:40:40.555Z"
	flow.Netflow.OctetTotalCount = int64(math.MaxUint32 - 1)
	flow.Netflow.PacketTotalCount = int64(math.MaxUint32 - 1)
	flow.Netflow.ProtocolIdentifier = protocols.TCP
	flow.Netflow.FlowEndReason = input.IdleTimeout
	flow.Netflow.Version = 10
	return flow
}

func getTestFlow3() *data.Flow {
	flow := &data.Flow{
		Host: "C",
	}
	flow.Netflow.SourceIPv4 = "3.3.3.3"
	flow.Netflow.SourcePort = 28972
	flow.Netflow.DestinationIPv4 = "4.4.4.4"
	flow.Netflow.DestinationPort = 443
	flow.Netflow.FlowStartMilliseconds = "2018-05-02T05:40:12.333Z"
	flow.Netflow.FlowEndMilliseconds = "2018-05-02T05:40:12.444Z"
	flow.Netflow.OctetTotalCount = math.MaxInt64
	flow.Netflow.PacketTotalCount = math.MaxInt64
	flow.Netflow.ProtocolIdentifier = protocols.TCP
	flow.Netflow.FlowEndReason = input.ActiveTimeout
	flow.Netflow.Version = 10
	return flow
}

func TestReader(t *testing.T) {
	fixtures := fixturesManager.BeginTest(t)
	defer fixturesManager.EndTest(t)
	env := fixtures.GetWithSkip(t, integrationtest.EnvironmentFixture.Key).(environment.Environment)
	inputDB := fixtures.GetWithSkip(t, inputDBTestFixture.Key).(mongodb.LogstashMongoInputDB)

	buff := mongodb.NewIDAtomicBuffer(inputDB.NewInputConnection(), env.Logger)
	reader := mongodb.NewReader(buff, 2*time.Second, env.Logger)

	c := inputDB.NewInputConnection()
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
	go func(flowTestResults chan<- testResult, flows <-chan input.Flow, wg *sync.WaitGroup) {
		f, ok := <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok := f.(*data.Flow)
		flowTestResults <- testResult{ok, "flow is *data.Flow", nil}

		flow1 := getTestFlow1()
		flow1.ID = outFlow.ID
		flowTestResults <- testResult{reflect.DeepEqual(flow1, outFlow), "flow1 read correctly", []interface{}{outFlow, flow1}}

		f, ok = <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok = f.(*data.Flow)
		flowTestResults <- testResult{ok, "flow is *data.Flow", nil}

		flow2 := getTestFlow2()
		flow2.ID = outFlow.ID
		flowTestResults <- testResult{reflect.DeepEqual(flow2, outFlow), "flow2 read correctly", []interface{}{outFlow, flow2}}

		f, ok = <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok = f.(*data.Flow)
		flowTestResults <- testResult{ok, "flow is *data.Flow", nil}

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
