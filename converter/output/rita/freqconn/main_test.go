package freqconn_test

import (
	"github.com/globalsign/mgo"
	"os"
	"testing"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/output/rita/freqconn"
)

const testThreshold = 10

const testDBName = "test"

const mongoContainerFixtureKey = "freqconn-test-db-container"

var fixtureManager *integrationtest.FixtureManager

var freqConnInitFixture = integrationtest.TestFixture{
	Key:         "freqconn-init",
	Requires:    []string{mongoContainerFixtureKey},
	LongRunning: true,
	Before: func(t *testing.T, fixtures integrationtest.FixtureData) (interface{}, bool) {
		//The tests maintain their own sessions, we just make sure the database is empty
		//and the collections have their proper indices
		mongoContainer := fixtures.Get(mongoContainerFixtureKey).(dbtest.MongoDBContainer)
		ssn, err := mongoContainer.NewSession()
		if err != nil {
			t.Error(err)
			return nil, false
		}

		connCollection := ssn.DB(testDBName).C(freqconn.ConnCollection)
		strobesCollection := ssn.DB(testDBName).C(freqconn.StrobesCollection)

		connCount, err := connCollection.Find(nil).Count()
		if err != nil {
			t.Error(err)
			return nil, false
		}

		if connCount != 0 {
			err = connCollection.DropCollection()
			if err != nil {
				t.Error(err)
				return nil, false
			}
		}

		strobesCount, err := strobesCollection.Find(nil).Count()
		if err != nil {
			t.Error(err)
			return nil, false
		}

		if strobesCount != 0 {
			err = strobesCollection.DropCollection()
			if err != nil {
				t.Error(err)
				return nil, false
			}
		}

		connIndices := []string{"$hashed:id_orig_h", "$hashed:id_resp_h", "-duration", "ts", "uid"}
		strobesIndices := []string{"$hashed:src", "$hashed:dst", "-connection_count"}

		for _, index := range connIndices {
			err := connCollection.EnsureIndex(mgo.Index{
				Key: []string{index},
			})
			if err != nil {
				t.Error(err)
				return nil, false
			}
		}

		for _, index := range strobesIndices {
			err := strobesCollection.EnsureIndex(mgo.Index{
				Key: []string{index},
			})
			if err != nil {
				t.Error(err)
				return nil, false
			}
		}
		ssn.Close()
		return nil, false
	},
}

//TestMain is responsible for setting up and tearing down any
//resources needed by all tests
func TestMain(m *testing.M) {
	fixtureManager = integrationtest.NewFixtureManager()
	fixtureManager.RegisterFixture(integrationtest.DockerLoaderFixture)
	fixtureManager.RegisterFixture(
		integrationtest.NewMongoDBContainerFixture(mongoContainerFixtureKey),
	)
	fixtureManager.RegisterFixture(freqConnInitFixture)
	fixtureManager.BeginTestPackage()
	returnCode := m.Run()
	fixtureManager.EndTestPackage()
	os.Exit(returnCode)
}
