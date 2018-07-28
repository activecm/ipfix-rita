package buffered_test

import (
	"os"
	"testing"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
)

const testDBName = "test"
const testCollectionName = "BUFFERED_TEST_COLLECTION"

const mongoContainerFixtureKey = "buffer-test-db-container"

var fixtureManager *integrationtest.FixtureManager

var testCollectionCleanupFixture = integrationtest.TestFixture{
	Key:         testCollectionName + "-cleanup",
	Requires:    []string{mongoContainerFixtureKey},
	LongRunning: true,
	After: func(t *testing.T, fixtures integrationtest.FixtureData) (interface{}, bool) {
		//The tests maintain their own sessions, we just make sure the database is empty
		//rather than closing the sessions
		mongoContainer := fixtures.Get(mongoContainerFixtureKey).(dbtest.MongoDBContainer)
		ssn, err := mongoContainer.NewSession()
		if err != nil {
			t.Error(err)
			return nil, false
		}
		coll := ssn.DB(testDBName).C(testCollectionName)
		count, err := coll.Count()
		if count != 0 && err == nil {
			coll.DropCollection()
		}
		ssn.Close()
		return nil, false
	},
}

//TestMain is responsible for setting up and tearing down any
//resources needed by all tests
func TestMain(m *testing.M) {
	fixtureManager = integrationtest.NewFixtureManager()
	fixtureManager.RegisterFixture(integrationtest.EnvironmentFixture)
	fixtureManager.RegisterFixture(integrationtest.DockerLoaderFixture)
	fixtureManager.RegisterFixture(
		integrationtest.NewMongoDBContainerFixture(mongoContainerFixtureKey),
	)
	fixtureManager.RegisterFixture(testCollectionCleanupFixture)
	fixtureManager.BeginTestPackage()
	returnCode := m.Run()
	fixtureManager.EndTestPackage()
	os.Exit(returnCode)
}
