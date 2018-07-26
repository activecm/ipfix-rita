package output

import (
	"github.com/activecm/ipfix-rita/converter/stitching/session"
)

//SessionWriter writes session aggregates to their final destination
type SessionWriter interface {
	//Write writes the session aggregate from the
	//input channel to their final destination
	Write(<-chan *session.Aggregate) <-chan error
}

//NullSessionWriter drops session aggregates so as to run the
//pipeline leading to it at full speed
type NullSessionWriter struct{}

func (n NullSessionWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	errs := make(chan error)
	go func() {
		for _ = range sessions {

		}
		close(errs)
	}()
	return errs
}
