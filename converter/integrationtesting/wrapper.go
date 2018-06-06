package integrationtesting

import (
	"context"
	"testing"

	"github.com/activecm/ipfix-rita/converter/environment"
)

//SetupIntegrationTest creates a new MongoDB container using Docker,
//sets up an appropriate testing Environment, and returns the Environment
//and a function for cleaning up the Docker image.
func SetupIntegrationTest(t *testing.T) (environment.Environment, func()) {
	loader, mongoDB := NewMongoDBContainer(t)
	env := NewIntegrationTestingEnvironment(t, mongoDB.GetMongoDBURI())
	return env, func() { loader.StopService(context.Background(), mongoDB) }
}
