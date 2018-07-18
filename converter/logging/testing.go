package logging

import (
	"testing"
)

type testLogger struct {
	t *testing.T
}

//NewTestLogger routes log calls to go test
func NewTestLogger(t *testing.T) Logger {
	return testLogger{t: t}
}

func (l testLogger) Error(err error, fields Fields) {
	if fields != nil {
		l.t.Errorf("%+v\n%+v\n", err, fields)
	} else {
		l.t.Errorf("%+v\n", err)
	}
}

func (l testLogger) Info(msg string, fields Fields) {
	if fields != nil {
		l.t.Logf("%s\n%+v\n", msg, fields)
	} else {
		l.t.Log(msg)
	}
}

func (l testLogger) Warn(msg string, fields Fields) {
	if fields != nil {
		l.t.Logf("%s\n%+v\n", msg, fields)
	} else {
		l.t.Log(msg)
	}
}
