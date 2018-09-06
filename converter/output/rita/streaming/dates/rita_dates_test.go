package dates_test

import (
	"testing"
	"time"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/output/rita"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestOutOfPeriodSessionsInGracePeriod(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts out in grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)
	currDBTime := clock.Now()
	prevDBTime := clock.Now().Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	targetTime := clock.Now().Add(-2 * time.Duration(intervalLengthMillis) * time.Millisecond)

	sessionChan := make(chan *session.Aggregate, bufferSize)
	sessions := generateNSessions(bufferSize, targetTime)
	for i := range sessions {
		sessionChan <- &sessions[i]
	}
	close(sessionChan)
	errs := ritaWriter.Write(sessionChan)

	mongoContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)
	ssn, err := mongoContainer.NewSession()
	if err != nil {
		t.Fatal(err)
	}

	for err = range errs {
		t.Fatal(err)
	}

	env := fixtures.Get(integrationtest.EnvironmentFixture.Key).(environment.Environment)
	currDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + currDBTime.Format(timeFormatString)
	prevDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + prevDBTime.Format(timeFormatString)
	targetDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + prevDBTime.Format(timeFormatString)

	currDBCount, err := ssn.DB(currDBName).C(rita.RitaConnInputCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, currDBCount)

	prevDBCount, err := ssn.DB(prevDBName).C(rita.RitaConnInputCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, prevDBCount)

	targetDBCount, err := ssn.DB(targetDBName).C(rita.RitaConnInputCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, targetDBCount)
	ssn.Close()
}

//TODO: simulate sessions over time
//TODO: create random sessions
//TODO: control in/ vs out of grace period

//To be working correctly:
//The current time is used to set the previous and current collections
//The data comes in on sessions channel
//The data is diverted towards the previous, current, or neither collection based on timestamp
//If the data's timestamp doesn't match the current collection
//The data is inserted into a buffer
//Data is buffered. If the buffer is full, the data is flushed on the write thread
//

//Test Cases:
//Open, send 1 current

func generateNSessions(n int64, targetSessionEnd time.Time) []session.Aggregate {
	targetSessionEndMillis := targetSessionEnd.UnixNano() / 1000000

	var sessions []session.Aggregate
	for i := int64(0); i < n; i++ {

		var sessA session.Aggregate
		var sessB session.Aggregate

		//create a mock flow
		a := input.NewFlowMock()
		b := &input.FlowMock{}
		//set the targeted session end timestamp
		a.MockFlowEndMilliseconds = targetSessionEndMillis
		a.MockFlowStartMilliseconds = a.MockFlowEndMilliseconds - 10*1000

		//Fill out b
		*b = *a
		b.MockDestinationIPAddress = a.MockSourceIPAddress
		b.MockSourceIPAddress = a.MockDestinationIPAddress
		b.MockDestinationPort = a.MockSourcePort
		b.MockSourcePort = a.MockDestinationPort

		session.FromFlow(a, &sessA)
		session.FromFlow(b, &sessB)
		sessA.Merge(&sessB)

		//guarantee we don't allow the same source/ dest ip pair
		provenUnique := false
		for !provenUnique {
			provenUnique = true

			for j := range sessions {
				if sessA.IPAddressA == sessions[j].IPAddressA && sessA.IPAddressB == sessions[j].IPAddressB {
					provenUnique = false
					a = input.NewFlowMock()
					a.MockFlowEndMilliseconds = targetSessionEndMillis
					a.MockFlowStartMilliseconds = a.MockFlowEndMilliseconds - 10*1000
					b = &input.FlowMock{}
					*b = *a
					b.MockDestinationIPAddress = a.MockSourceIPAddress
					b.MockSourceIPAddress = a.MockDestinationIPAddress
					b.MockDestinationPort = a.MockSourcePort
					b.MockSourcePort = a.MockDestinationPort

					sessA.Clear()
					sessB.Clear()
					session.FromFlow(a, &sessA)
					session.FromFlow(b, &sessB)
					sessA.Merge(&sessB)
					break
				}
			}
		}

		sessions = append(sessions, sessA)
	}
	return sessions
}
