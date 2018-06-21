package output

import (
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/davecgh/go-spew/spew"
)

//SessionWriter writes session aggregates to their final destination
type SessionWriter interface {
	//Write writes the session aggregate to its final destination
	Write(sess *session.Aggregate) error
}

//SpewRITAConnWriter writes session aggregates out to the terminal
//as RITA conn objects
type SpewRITAConnWriter struct{}

//Write spews the session to the terminal as a RITA Conn object
func (s SpewRITAConnWriter) Write(sess *session.Aggregate) error {
	var conn parsetypes.Conn
	sess.ToRITAConn(&conn, func(ipAddress string) bool { return false })
	spew.Println(conn)
	return nil
}
