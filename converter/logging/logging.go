package logging

import (
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

//Fields holds arbitrary data for structured logging
type Fields map[string]interface{}

//Logger provides the means to log errors, status updates, etc.
type Logger interface {
	Error(err error, fields Fields)
	Info(msg string, fields Fields)
	Warn(msg string, fields Fields)
}

//stackTracer is defined as per https://godoc.org/github.com/pkg/errors#Cause
type stackTracer interface {
	StackTrace() errors.StackTrace
}

//logrusLogger implements Logger by value (since its just a pointer)
type logrusLogger struct {
	logger *log.Logger
}

//NewLogrusLogger creates a Logger backed by the logrus logging system
func NewLogrusLogger() Logger {
	log := logrusLogger{
		logger: log.New(),
	}

	//Send logs to stdout rather than stderr
	//as the logger will be the only terminal output
	log.logger.Out = os.Stdout
	return log
}

func (l logrusLogger) Error(err error, fields Fields) {
	if fields == nil {
		fields = make(Fields)
	}
	if err, ok := err.(stackTracer); ok {
		fields["stacktrace"] = err.StackTrace()
	}
	l.logger.WithFields(log.Fields(fields)).Error(err.Error())
}

func (l logrusLogger) Info(msg string, fields Fields) {
	l.logger.WithFields(log.Fields(fields)).Info(msg)
}

func (l logrusLogger) Warn(msg string, fields Fields) {
	l.logger.WithFields(log.Fields(fields)).Warn(msg)
}
