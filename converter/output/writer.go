package output

import (
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/davecgh/go-spew/spew"
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

//SpewRITAConnWriter writes session aggregates out to the terminal
//as RITA conn objects
type SpewRITAConnWriter struct{}

//Write spews the sessions to the terminal as a RITA Conn object
func (s SpewRITAConnWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	errs := make(chan error)
	go func() {
		for sess := range sessions {
			var conn parsetypes.Conn
			sess.ToRITAConn(&conn, func(ipAddress string) bool { return false })
			spew.Println(conn)
		}
		close(errs)
	}()
	return errs
}
