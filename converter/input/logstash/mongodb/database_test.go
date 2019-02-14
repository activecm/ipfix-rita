package mongodb_test

import (
	"testing"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/input/logstash/mongodb"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
)

//inputDBContainerTestFixture inserts a valid LogstashMongoInputDB
//into the testing environment backed by Docker
var inputDBContainerTestFixtureKey = "inputDB-Container"

var inputDBTestFixture = integrationtest.TestFixture{
	Key:         "inputDB",
	LongRunning: true,
	Requires: []string{
		inputDBContainerTestFixtureKey,
		integrationtest.EnvironmentFixture.Key,
	},
	Before: func(t *testing.T, fixtures integrationtest.FixtureData) (interface{}, bool) {
		mongoContainer := fixtures.Get(inputDBContainerTestFixtureKey).(dbtest.MongoDBContainer)
		env := fixtures.Get(integrationtest.EnvironmentFixture.Key).(environment.Environment)

		//Not a fan of busting through the config interface to set the data
		//but the interface is immutable (and should be during normal operation)
		//TODO: Add mutators to the MongoDB config interface
		testLogstashConfig := env.GetInputConfig().GetLogstashMongoDBConfig()
		testMongoConfig := testLogstashConfig.GetConnectionConfig().(*integrationtest.MongoDBConfig)
		testMongoConfig.SetConnectionString(mongoContainer.GetMongoDBURI())

		//Note: the env stored in the fixture will be changed as well
		//since the config field on environment is a pointer

		inputDB, err := mongodb.NewLogstashMongoInputDB(testLogstashConfig)
		if err != nil {
			panic(err)
		}
		return inputDB, true
	},
	After: func(t *testing.T, fixtures integrationtest.FixtureData) (interface{}, bool) {
		inputDB := fixtures.Get("inputDB").(mongodb.LogstashMongoInputDB)
		inputDB.Close()
		return nil, true
	},
}
