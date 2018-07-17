package mgologstash

import (
	"context"
	"time"

	"github.com/activecm/ipfix-rita/converter/ipfix"
	"github.com/pkg/errors"
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
	errs := make(chan error)

	go func(buffer Buffer, pollWait time.Duration, out chan<- ipfix.Flow, errs chan<- error) {
		pollTicker := time.NewTicker(pollWait)

		r.drainInner(ctx, buffer, out, errs)
	Loop:
		for {
			select {
			case <-ctx.Done():
				errs <- errors.Wrap(ctx.Err(), "reading from input buffer cancelled")
				break Loop
			case <-pollTicker.C:
				r.drainInner(ctx, buffer, out, errs)
			}
		}
		buffer.Close()
		pollTicker.Stop()
		close(errs)
		close(out)
	}(r.buffer, r.pollWait, out, errs)

	return out, errs
}

func (r Reader) drainInner(ctx context.Context, buffer Buffer, out chan<- ipfix.Flow, errs chan<- error) {
	flow := &Flow{}
	for buffer.Next(flow) {
		out <- flow
		//ensure we stop even if there is more data
		if ctx.Err() != nil {
			errs <- errors.Wrap(ctx.Err(), "reading from input buffer cancelled")
			break
		}
		flow = &Flow{}
	}
	if buffer.Err() != nil {
		errs <- errors.Wrap(buffer.Err(), "could not drain input buffer")
	}
}
