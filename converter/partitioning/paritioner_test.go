package partitioning

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/activecm/ipfix-rita/converter/protocols"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const HostA = "1.1.1.1"
const HostB = "2.2.2.2"
const HostC = "3.3.3.3"
const PortA = 11111
const PortB = 22222

func TestSelectPartition(t *testing.T) {
	rand.Seed(0)

	p := NewHashPartitioner(5, 0)
	var partitions [5][]ipfix.Flow
	for i := range partitions {
		partitions[i] = make([]ipfix.Flow, 0)
	}

	//generate a bunch of data
	testData := make([]*ipfix.FlowMock, 0)
	for i := 0; i < 1000; i++ {
		testData = append(testData, ipfix.NewFlowMock())
	}

	//use the first 100 to make sure same key hashes to same bin
	for i := 0; i < 100; i++ {
		testData[i].MockSourceIPAddress = HostA
		testData[i].MockDestinationIPAddress = HostB
		testData[i].MockExporter = HostC
		testData[i].MockProtocolIdentifier = protocols.TCP
		testData[i].MockSourcePort = PortA
		testData[i].MockDestinationPort = PortB
	}
	firstBin := p.selectPartition(testData[0])

	for i := 0; i < 100; i++ {
		bin := p.selectPartition(testData[i])
		require.Equal(t, firstBin, bin)
	}

	//use the remaining 900 to make sure the flows are split roughly equally
	for i := 100; i < 1000; i++ {
		bin := p.selectPartition(testData[i])
		partitions[bin] = append(partitions[bin], testData[i])
	}
	expected := (1000 - 100) / 5
	delta := 25
	for i, partition := range partitions {
		t.Logf("Partition %d: %d flows\n", i, len(partition))
		require.Condition(t, assert.Comparison(func() bool {
			return expected-delta <= len(partition) && len(partition) <= expected+delta
		}))
	}
}

func TestPartition(t *testing.T) {
	rand.Seed(0)
	p := NewHashPartitioner(5, 0)
	inputChan := make(chan ipfix.Flow)
	//generate a bunch of data
	go func() {
		for i := 0; i < 1000; i++ {
			inputChan <- ipfix.NewFlowMock()
		}
		close(inputChan)
	}()

	outChans, _ := p.Partition(context.Background(), inputChan)
	require.Len(t, outChans, 5)

	var counts [5]int
	wg := sync.WaitGroup{}
	wg.Add(5)
	t.Log("Kicking off readers")

	for i := range outChans {
		go func(outChan <-chan ipfix.Flow, counter *int) {
			for _ = range outChan {
				*counter++
			}
			wg.Done()
		}(outChans[i], &counts[i])
	}

	wg.Wait()
	expected := 1000 / 5
	delta := 25
	for i, count := range counts {
		t.Logf("Partition %d: %d flows\n", i, count)
		require.Condition(t, assert.Comparison(func() bool {
			return expected-delta <= count && count <= expected+delta
		}))
	}
}
