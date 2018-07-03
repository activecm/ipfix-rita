package environmenttest

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/environment"
)

//NewIntegrationTestingEnvironment creates a new environment.Environment
//suitable for testing. Initializes Environment.DB with the given MongoDB URI.
//MongoDB must be run without encryption/ authentication.
func NewIntegrationTestingEnvironment(t *testing.T, mongoDBURI string) environment.Environment {
	envOut := environment.Environment{
		Config: newTestingConfig(mongoDBURI),
		Logger: newTestingLogger(t),
	}
	var err error
	envOut.DB, err = database.NewDB(envOut.GetMongoDBConfig(), envOut.GetRITAConfig())
	if err != nil {
		envOut.Error(err, nil)
		t.FailNow()
	}
	return envOut
}
