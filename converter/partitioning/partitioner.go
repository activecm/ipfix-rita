package partitioning

import (
	"context"
	"encoding/binary"
	"hash/fnv"

	"github.com/activecm/ipfix-rita/converter/ipfix"
)

//HashPartitioner partitions an input stream of IPFIX Flow data
//into N partitions such that Flows with the same Flow Key are
//sent to the same partition
type HashPartitioner struct {
	numPartitions       int32
	partitionBufferSize int
}

//NewHashPartitioner is a convenience function for creating a HashPartitioner
func NewHashPartitioner(numPartitions int32, partitionBufferSize int) HashPartitioner {
	return HashPartitioner{
		numPartitions:       numPartitions,
		partitionBufferSize: partitionBufferSize,
	}
}

//Partition splits a channel of IPFIX Flow data
//across several channels, ensuring that two Flows with the same
//flow key are not processed at the same time.
func (p HashPartitioner) Partition(ctx context.Context,
	input <-chan ipfix.Flow) ([]<-chan ipfix.Flow, <-chan error) {

	errs := make(chan error)
	partitions := make([]chan ipfix.Flow, p.numPartitions)
	outPartitions := make([]<-chan ipfix.Flow, p.numPartitions)
	for i := range partitions {
		partitions[i] = make(chan ipfix.Flow, p.partitionBufferSize)
		outPartitions[i] = partitions[i]
	}

	go p.pump(ctx, input, partitions, errs)

	return outPartitions, errs
}

func (p HashPartitioner) pump(ctx context.Context, input <-chan ipfix.Flow,
	partitions []chan ipfix.Flow, errs chan<- error) {

Loop:
	for {
		select {
		case in, ok := <-input:
			if !ok {
				break Loop
			}
			partitions[p.selectPartition(in)] <- in
		case <-ctx.Done():
			errs <- ctx.Err()
			break Loop
		}
	}
	for i := range partitions {
		close(partitions[i])
	}
}

func (p HashPartitioner) selectPartition(f ipfix.Flow) int32 {
	hasher := fnv.New32()
	hasher.Write([]byte(f.Exporter()))
	hasher.Write([]byte(f.SourceIPAddress()))
	hasher.Write([]byte(f.DestinationIPAddress()))

	var buffer [2]byte
	bufferSlice := buffer[:]

	binary.LittleEndian.PutUint16(bufferSlice, f.SourcePort())
	hasher.Write(bufferSlice)

	binary.LittleEndian.PutUint16(bufferSlice, f.DestinationPort())
	hasher.Write(bufferSlice)

	bufferSlice = buffer[:1]
	bufferSlice[0] = uint8(f.ProtocolIdentifier())
	hasher.Write(bufferSlice)

	partition := int32(hasher.Sum32()) % p.numPartitions
	if partition < 0 {
		partition = -partition
	}
	return partition
}
