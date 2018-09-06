package dates_test

import (
	"os"
	"testing"
	"time"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/output/rita/streaming/dates"
	"github.com/benbjohnson/clock"
)

var fixtureManager *integrationtest.FixtureManager

const mongoContainerFixtureKey = "date-test-db-container"

const bufferSize = int64(5)
const autoFlushTime = 5 * time.Second
const intervalLengthMillis = int64(30 * 1000)
const gracePeriodCutoffMillis = int64(10 * 1000)
const timeFormatString = "Jan-_2-15:04:05-000"

var clockFixture = integrationtest.TestFixture{
	Key:         "clock",
	LongRunning: false,
	Requires:    []string{},
	Before: func(t *testing.T, fixtures integrationtest.FixtureData) (interface{}, bool) {
		clock := clock.NewMock()
		return clock, true
	},
}

var streamingRITATimeIntervalWriterFixture = integrationtest.TestFixture{
	Key:         "streamingRITATimeIntervalWriter",
	LongRunning: true,
	Requires: []string{
		mongoContainerFixtureKey,
		integrationtest.EnvironmentFixture.Key,
		clockFixture.Key,
	},
	Before: func(t *testing.T, fixtures integrationtest.FixtureData) (interface{}, bool) {
		mongoContainer := fixtures.Get(mongoContainerFixtureKey).(dbtest.MongoDBContainer)
		env := fixtures.Get(integrationtest.EnvironmentFixture.Key).(environment.Environment)
		clock := fixtures.Get(clockFixture.Key).(clock.Clock)
		//Not a fan of busting through the config interface to set the data
		//but the interface is immutable (and should be during normal operation)
		//TODO: Add mutators to the MongoDB config interface
		testOutputConfig := env.GetOutputConfig().GetRITAConfig().GetConnectionConfig().(*integrationtest.MongoDBConfig)
		testOutputConfig.SetConnectionString(mongoContainer.GetMongoDBURI())

		ritaWriter, err := dates.NewStreamingRITATimeIntervalWriter(
			env.GetOutputConfig().GetRITAConfig(),
			env.GetIPFIXConfig(),
			bufferSize, autoFlushTime,
			intervalLengthMillis, gracePeriodCutoffMillis,
			clock, timeFormatString,
			env.Logger,
		)

		if err != nil {
			return nil, false
		}

		return ritaWriter, true
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
	fixtureManager.RegisterFixture(clockFixture)
	fixtureManager.RegisterFixture(streamingRITATimeIntervalWriterFixture)
	fixtureManager.BeginTestPackage()
	returnCode := m.Run()
	fixtureManager.EndTestPackage()
	os.Exit(returnCode)
}
