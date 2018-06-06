package environmenttest

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/logging"
)

type testingLogger struct {
	t *testing.T
}

//newTestingLogger routes log calls to go test
func newTestingLogger(t *testing.T) logging.Logger {
	return testingLogger{t: t}
}

func (l testingLogger) Error(err error, fields logging.Fields) {
	l.t.Errorf("%s\n%+v\n", err, fields)
}

func (l testingLogger) Info(msg string, fields logging.Fields) {
	l.t.Logf("%s\n%+v\n", msg, fields)
}

func (l testingLogger) Warn(msg string, fields logging.Fields) {
	l.t.Logf("%s\n%+v\n", msg, fields)
}
