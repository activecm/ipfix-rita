package logging

import (
	"testing"
)

//TestLogger implements logging.Logger with
//the methods provided by go test's testing.T
type TestLogger struct {
	t *testing.T
}

//NewTestLogger routes log calls to go test
func NewTestLogger(t *testing.T) Logger {
	return TestLogger{t: t}
}

//Error displays an error and all of its inner fields along
//with arbitrary data as specified in Fields
func (l TestLogger) Error(err error, fields Fields) {
	if fields != nil {
		l.t.Errorf("%+v\n%+v\n", err, fields)
	} else {
		l.t.Errorf("%+v\n", err)
	}
}

//Info displays a message and all of its inner fields along
//with arbitrary data as specified in Fields
func (l TestLogger) Info(msg string, fields Fields) {
	if fields != nil {
		l.t.Logf("%s\n%+v\n", msg, fields)
	} else {
		l.t.Log(msg)
	}
}

//Warn displays a message and all of its inner fields along
//with arbitrary data as specified in Fields
func (l TestLogger) Warn(msg string, fields Fields) {
	if fields != nil {
		l.t.Logf("%s\n%+v\n", msg, fields)
	} else {
		l.t.Log(msg)
	}
}
