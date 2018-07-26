package rita

import (
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/davecgh/go-spew/spew"
)

//Only used for debugging.

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
			spew.Dump(conn)
		}
		close(errs)
	}()
	return errs
}
