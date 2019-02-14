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

func newTestFlow(
	host string, sourceIP string, sourcePort uint16, sourceV6 bool,
	destinationIP string, destinationPort uint16, destinationV6 bool,
	flowStart string, flowEnd string, octetCount int64, packetCount int64,
	protocol protocols.Identifier, endReason input.FlowEndReason, version uint8,
) *data.Flow {

	flow := &data.Flow{
		Host: host,
	}
	if sourceV6 {
		flow.Netflow.SourceIPv6 = sourceIP
	} else {
		flow.Netflow.SourceIPv4 = sourceIP
	}

	if destinationV6 {
		flow.Netflow.DestinationIPv6 = destinationIP
	} else {
		flow.Netflow.DestinationIPv4 = destinationIP
	}
	flow.Netflow.SourcePort = sourcePort
	flow.Netflow.DestinationPort = destinationPort
	flow.Netflow.FlowStartMilliseconds = flowStart
	flow.Netflow.FlowEndMilliseconds = flowEnd
	flow.Netflow.OctetTotalCount = octetCount
	flow.Netflow.PacketTotalCount = packetCount
	flow.Netflow.ProtocolIdentifier = protocol
	flow.Netflow.FlowEndReason = endReason
	flow.Netflow.Version = version
	return flow
}

var testFlow1 = newTestFlow(
	"A", "1.1.1.1", 24846, false,
	"2.2.2.2", 53, false,
	"2018-05-04T22:36:40.766Z", "2018-05-04T22:36:40.960Z",
	int64(math.MaxUint32+1), int64(math.MaxUint32+1),
	protocols.UDP, input.ActiveTimeout, 10,
)

var testFlow2 = newTestFlow(
	"B", "1.1.1.1", 24846, false,
	"2.2.2.2", 53, false,
	"2018-05-04T22:40:40.555Z", "2018-05-04T22:40:40.555Z",
	int64(math.MaxUint32-1), int64(math.MaxUint32-1),
	protocols.TCP, input.IdleTimeout, 10,
)

var testFlow3 = newTestFlow(
	"C", "3.3.3.3", 28972, false,
	"4.4.4.4", 443, false,
	"2018-05-02T05:40:12.333Z", "2018-05-02T05:40:12.444Z",
	math.MaxInt64, math.MaxInt64,
	protocols.TCP, input.ActiveTimeout, 10,
)

func TestReader(t *testing.T) {
	fixtures := fixturesManager.BeginTest(t)
	defer fixturesManager.EndTest(t)
	env := fixtures.GetWithSkip(t, integrationtest.EnvironmentFixture.Key).(environment.Environment)
	inputDB := fixtures.GetWithSkip(t, inputDBTestFixture.Key).(mongodb.LogstashMongoInputDB)

	buff := mongodb.NewIDAtomicBuffer(inputDB.NewInputConnection(), env.Logger)
	reader := mongodb.NewReader(buff, 2*time.Second, env.Logger)

	c := inputDB.NewInputConnection()
	err := c.Insert(testFlow1)
	require.Nil(t, err)
	err = c.Insert(testFlow2)
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

		flow1 := &data.Flow{}
		*flow1 = *testFlow1
		flow1.ID = outFlow.ID
		flowTestResults <- testResult{reflect.DeepEqual(flow1, outFlow), "flow1 read correctly", []interface{}{outFlow, flow1}}

		f, ok = <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok = f.(*data.Flow)
		flowTestResults <- testResult{ok, "flow is *data.Flow", nil}

		flow2 := &data.Flow{}
		*flow2 = *testFlow2
		flow2.ID = outFlow.ID
		flowTestResults <- testResult{reflect.DeepEqual(flow2, outFlow), "flow2 read correctly", []interface{}{outFlow, flow2}}

		f, ok = <-flows
		flowTestResults <- testResult{ok, "flows available for reading", nil}
		outFlow, ok = f.(*data.Flow)
		flowTestResults <- testResult{ok, "flow is *data.Flow", nil}

		flow3 := &data.Flow{}
		*flow3 = *testFlow3
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
	err = c.Insert(testFlow3)
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
