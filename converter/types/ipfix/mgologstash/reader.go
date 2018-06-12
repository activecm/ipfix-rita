package mgologstash

import (
	"context"
	"time"

	"github.com/activecm/ipfix-rita/converter/types/ipfix"
)

//Reader implements ipfix.Reader
type Reader struct {
	buffer   Buffer
	pollWait time.Duration
}

//NewReader returns a new ipfix.Reader backed by a mgologstash.Buffer
func NewReader(buffer Buffer, pollWait time.Duration) ipfix.Reader {
	return Reader{
		buffer:   buffer,
		pollWait: pollWait,
	}
}

//Drain asynchronously drains a mgologstash.Buffer
func (r Reader) Drain(ctx context.Context) (<-chan ipfix.Flow, <-chan error) {
	out := make(chan ipfix.Flow)
	errors := make(chan error)

	go func(buffer Buffer, pollWait time.Duration, out chan<- ipfix.Flow, errors chan<- error) {
		pollTicker := time.NewTicker(pollWait)
	Loop:
		for {
			select {
			case <-ctx.Done():
				errors <- ctx.Err()
				break Loop
			case <-pollTicker.C:
				flow := &Flow{}
				for buffer.Next(flow) {
					out <- flow
					flow = &Flow{}
				}
				if buffer.Err() != nil {
					errors <- buffer.Err()
				}
			}
		}
		buffer.Close()
		close(errors)
		close(out)
	}(r.buffer, r.pollWait, out, errors)

	return out, errors
}
