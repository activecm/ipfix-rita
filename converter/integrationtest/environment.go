package integrationtest

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/environment"
)

//newEnvironment creates a new environment.Environment
//suitable for testing. Initializes Environment.DB with the given MongoDB URI.
//MongoDB must be run without encryption/ authentication.
func newEnvironment(t *testing.T, mongoDBURI string) environment.Environment {
	envOut := environment.Environment{
		Config: newConfig(mongoDBURI),
		Logger: newLogger(t),
	}
	var err error
	envOut.DB, err = database.NewDB(envOut.GetMongoDBConfig(), envOut.GetRITAConfig())
	if err != nil {
		envOut.Error(err, nil)
		t.FailNow()
	}
	return envOut
}
