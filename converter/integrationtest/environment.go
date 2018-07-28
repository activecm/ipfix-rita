package integrationtest

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/logging"
)

//newEnvironment creates a new environment.Environment
//suitable for testing. Initializes Environment.DB with the given MongoDB URI.
//MongoDB must be run without encryption/ authentication.
func newEnvironment(t *testing.T) environment.Environment {
	envOut := environment.Environment{
		Config: &TestConfig{},
		Logger: logging.NewTestLogger(t),
	}
	return envOut
}

//EnvironmentFixture ensures a proper testing environment
//is loaded into a FixtureManager.
var EnvironmentFixture = TestFixture{
	Key: "environment",
	Before: func(t *testing.T, data FixtureData) (interface{}, bool) {
		return newEnvironment(t), true
	},
}
