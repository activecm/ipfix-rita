package mgologstash

import (
	"context"
	"time"

	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/pkg/errors"
)

//Reader implements input.Reader
type Reader struct {
	buffer   Buffer
	pollWait time.Duration
	log      logging.Logger
}

//NewReader returns a new input.Reader backed by a mgologstash.Buffer.
//If there is no data in the buffer, the reader will wait for `pollWait`,
//then try reading the buffer again.
func NewReader(buffer Buffer, pollWait time.Duration, log logging.Logger) input.Reader {
	return Reader{
		buffer:   buffer,
		pollWait: pollWait,
		log:      log,
	}
}

//Drain asynchronously drains a mgologstash.Buffer
func (r Reader) Drain(ctx context.Context) (<-chan input.Flow, <-chan error) {
	out := make(chan input.Flow)
	errs := make(chan error)

	go func(buffer Buffer, pollWait time.Duration, out chan<- input.Flow, errs chan<- error) {
		pollTicker := time.NewTicker(pollWait)

		r.drainInner(ctx, buffer, out, errs)
	Loop:
		for {
			select {
			case <-ctx.Done():
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

func (r Reader) drainInner(ctx context.Context, buffer Buffer, out chan<- input.Flow, errs chan<- error) {
	r.log.Info("checking input buffer for more data", nil)
	dataFound := false
	flow := &Flow{}
	for buffer.Next(flow) {
		if !dataFound {
			r.log.Info("reading new data from input buffer", nil)
			dataFound = true
		}

		out <- flow
		//ensure we stop even if there is more data
		if ctx.Err() != nil {
			break
		}
		flow = &Flow{}
	}
	if !dataFound {
		r.log.Info("no data available in input buffer", nil)
	}
	if buffer.Err() != nil {
		errs <- errors.Wrap(buffer.Err(), "could not drain input buffer")
	}
	r.log.Info("waiting for more data to arrive in input buffer", nil)
}
