package mongodb_test

import (
	"os"
	"testing"

	"github.com/activecm/ipfix-rita/converter/integrationtest"
)

var fixturesManager *integrationtest.FixtureManager

//TestMain is responsible for setting up and tearing down any
//resources needed by all tests
func TestMain(m *testing.M) {
	fixturesManager = integrationtest.NewFixtureManager()
	//add the environment to the test fixtures
	fixturesManager.RegisterFixture(integrationtest.EnvironmentFixture)
	//add the docker loader to the test fixtures
	fixturesManager.RegisterFixture(integrationtest.DockerLoaderFixture)
	//add the mongo db container handle to the test fixtures
	fixturesManager.RegisterFixture(
		integrationtest.NewMongoDBContainerFixture(inputDBContainerTestFixtureKey),
	)
	//add the input database to the test fixtures
	fixturesManager.RegisterFixture(inputDBTestFixture)
	fixturesManager.BeginTestPackage()
	returnCode := m.Run()
	fixturesManager.EndTestPackage()
	os.Exit(returnCode)
}
