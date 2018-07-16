package integrationtest

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/logging"
)

type logger struct {
	t *testing.T
}

//newLogger routes log calls to go test
func newLogger(t *testing.T) logging.Logger {
	return logger{t: t}
}

func (l logger) Error(err error, fields logging.Fields) {
	l.t.Errorf("%s\n%+v\n", err, fields)
}

func (l logger) Info(msg string, fields logging.Fields) {
	l.t.Logf("%s\n%+v\n", msg, fields)
}

func (l logger) Warn(msg string, fields logging.Fields) {
	l.t.Logf("%s\n%+v\n", msg, fields)
}
