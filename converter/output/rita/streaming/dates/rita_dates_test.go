package dates_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/input"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/output/rita"
	"github.com/activecm/ipfix-rita/converter/output/rita/constants"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/benbjohnson/clock"
	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/require"
)

func TestOutOfPeriodSessionsInGracePeriod(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts out in grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)
	currDBTime := clock.Now().In(timezone)
	prevDBTime := clock.Now().In(timezone).Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	targetDBTime := clock.Now().In(timezone).Add(-2 * time.Duration(intervalLengthMillis) * time.Millisecond)

	sessionChan := make(chan *session.Aggregate, bufferSize)
	sessions := generateNSessions(bufferSize, targetDBTime)
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
	targetDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + targetDBTime.Format(timeFormatString)

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, currDBCount)

	prevDBCount, err := ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, prevDBCount)

	targetDBCount, err := ssn.DB(targetDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, targetDBCount)
	ssn.Close()
}

func TestOutOfPeriodSessionsOutOfGracePeriod(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)

	//don't adjust clock so db names align with intervals
	currDBTime := clock.Now().In(timezone)
	prevDBTime := clock.Now().In(timezone).Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	targetDBTime := clock.Now().In(timezone).Add(-2 * time.Duration(intervalLengthMillis) * time.Millisecond)

	//clock starts outside of the grace period
	clock.Add(time.Duration(gracePeriodCutoffMillis) * time.Millisecond)

	sessionChan := make(chan *session.Aggregate, bufferSize)
	sessions := generateNSessions(bufferSize, targetDBTime)
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
	targetDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + targetDBTime.Format(timeFormatString)

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, currDBCount)

	prevDBCount, err := ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, prevDBCount)

	targetDBCount, err := ssn.DB(targetDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, targetDBCount)
	ssn.Close()
}

func TestPreviousPeriodSessionsInGracePeriod(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts out in grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)
	currDBTime := clock.Now().In(timezone)
	prevDBTime := clock.Now().In(timezone).Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)

	sessionChan := make(chan *session.Aggregate, bufferSize)
	sessions := generateNSessions(bufferSize, prevDBTime)
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

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, currDBCount)

	fmt.Printf("Checking %s.%s", prevDBName, constants.ConnCollection)
	prevDBCount, err := ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), prevDBCount)

	ssn.Close()
}

func TestPreviousPeriodSessionsOutOfGracePeriod(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts outside of the grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)

	//don't adjust clock so db names align with intervals
	currDBTime := clock.Now().In(timezone)
	prevDBTime := clock.Now().In(timezone).Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	targetDBTime := prevDBTime

	//clock starts outside of the grace period
	clock.Add(time.Duration(gracePeriodCutoffMillis) * time.Millisecond)

	sessionChan := make(chan *session.Aggregate, bufferSize)

	//test snapping dbnames to interval times
	targetDBTimeWithOffset := targetDBTime.Add(time.Duration(gracePeriodCutoffMillis/2) * time.Millisecond)
	sessions := generateNSessions(bufferSize, targetDBTimeWithOffset)
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
	targetDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + targetDBTime.Format(timeFormatString)

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, currDBCount)

	prevDBCount, err := ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, prevDBCount)

	targetDBCount, err := ssn.DB(targetDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, targetDBCount)
	ssn.Close()
}

func TestCurrentPeriodSessionsInGracePeriod(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts out in grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)
	currDBTime := clock.Now().In(timezone)
	prevDBTime := clock.Now().In(timezone).Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	targetDBTime := currDBTime

	sessionChan := make(chan *session.Aggregate, bufferSize)
	sessions := generateNSessions(bufferSize, targetDBTime)
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
	targetDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + targetDBTime.Format(timeFormatString)

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), currDBCount)

	prevDBCount, err := ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, prevDBCount)

	targetDBCount, err := ssn.DB(targetDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), targetDBCount)
	ssn.Close()
}

func TestCurrentPeriodSessionsOutOfGracePeriod(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts outside of the grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)

	//don't adjust clock so db names align with intervals
	currDBTime := clock.Now().In(timezone)
	prevDBTime := clock.Now().In(timezone).Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	targetDBTime := currDBTime

	//clock starts outside of the grace period
	clock.Add(time.Duration(gracePeriodCutoffMillis) * time.Millisecond)

	sessionChan := make(chan *session.Aggregate, bufferSize)

	//test snapping dbnames to interval times
	targetDBTimeWithOffset := targetDBTime.Add(time.Duration(gracePeriodCutoffMillis/2) * time.Millisecond)
	sessions := generateNSessions(bufferSize, targetDBTimeWithOffset)
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
	targetDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + targetDBTime.Format(timeFormatString)

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), currDBCount)

	prevDBCount, err := ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, prevDBCount)

	targetDBCount, err := ssn.DB(targetDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), targetDBCount)
	ssn.Close()
}

func TestGracePeriodFlip(t *testing.T) {
	//If this test fails, its probably because of the bad waits needed
	//Ideally these would be replaced with a time-bounded check loop
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)

	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)
	currDBTime := clock.Now().In(timezone)
	nextDBTime := clock.Now().In(timezone).Add(1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	prevDBTime := clock.Now().In(timezone).Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)

	env := fixtures.Get(integrationtest.EnvironmentFixture.Key).(environment.Environment)
	currDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + currDBTime.Format(timeFormatString)
	nextDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + nextDBTime.Format(timeFormatString)
	prevDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + prevDBTime.Format(timeFormatString)

	//get the mongo session ready for checking
	mongoContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)
	ssn, err := mongoContainer.NewSession()
	if err != nil {
		t.Fatal(err)
	}

	sessionChan := make(chan *session.Aggregate, bufferSize)

	errChan := ritaWriter.Write(sessionChan)
	var errs []error
	go func() {
		for err := range errChan {
			errs = append(errs, err)
		}
	}()

	//We need to wait for asynchronous operations to finish at several
	//points in the test.
	waitTime := 10 * time.Second

	prevSessions := generateNSessions(bufferSize, prevDBTime)

	for i := range prevSessions {
		sessionChan <- &prevSessions[i]
	}

	//Wait for buffer to clear. This might cause the test to fail on
	//slow machines
	time.Sleep(waitTime)

	prevDBCount, err := ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), prevDBCount)

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, currDBCount)

	nextDBCount, err := ssn.DB(nextDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, nextDBCount)

	//This should advance up past the grace period
	clock.Add(time.Duration(gracePeriodCutoffMillis) * time.Millisecond)

	time.Sleep(waitTime)

	for i := range prevSessions {
		sessionChan <- &prevSessions[i]
	}

	//Wait for buffer to clear. This might cause the test to fail on
	//slow machines
	time.Sleep(waitTime)

	prevDBCount, err = ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), prevDBCount)

	currDBCount, err = ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, currDBCount)

	nextDBCount, err = ssn.DB(nextDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, nextDBCount)

	//This should advance us to the next time segment.
	//prev will now be outside of scope, curr will be prev, and next will be curr
	clock.Add(time.Duration(intervalLengthMillis-gracePeriodCutoffMillis) * time.Millisecond)

	//Wait for changes to take place
	time.Sleep(waitTime)

	newPrevOldCurrSessions := generateNSessions(bufferSize, currDBTime)
	for i := range newPrevOldCurrSessions {
		sessionChan <- &newPrevOldCurrSessions[i]
	}

	//Wait for buffer to clear. This might cause the test to fail on
	//slow machines
	time.Sleep(waitTime)

	prevDBCount, err = ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), prevDBCount)

	currDBCount, err = ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), currDBCount)

	nextDBCount, err = ssn.DB(nextDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 0, nextDBCount)

	newCurrOldNextSessions := generateNSessions(bufferSize, nextDBTime)
	for i := range newCurrOldNextSessions {
		sessionChan <- &newCurrOldNextSessions[i]
	}

	//Wait for buffer to clear. This might cause the test to fail on
	//slow machines
	time.Sleep(waitTime)

	prevDBCount, err = ssn.DB(prevDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), prevDBCount)

	currDBCount, err = ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), currDBCount)

	nextDBCount, err = ssn.DB(nextDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, int(bufferSize), nextDBCount)

	close(sessionChan)

	for i := range errs {
		t.Fatal(errs[i])
	}
}

func TestBufferFlushOnClose(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts out in grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)
	currDBTime := clock.Now().In(timezone)

	sessionChan := make(chan *session.Aggregate, bufferSize)
	sessions := generateNSessions(bufferSize-1, currDBTime)
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

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 4, currDBCount)
	ssn.Close()
}

func TestBufferFlushOnTimeout(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)
	//clock starts out in grace period
	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)
	currDBTime := clock.Now().In(timezone)

	mongoContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)
	ssn, err := mongoContainer.NewSession()
	if err != nil {
		t.Fatal(err)
	}

	sessionChan := make(chan *session.Aggregate, bufferSize)
	errChan := ritaWriter.Write(sessionChan)
	var errs []error
	go func() {
		for err := range errChan {
			errs = append(errs, err)
		}
	}()

	sessions := generateNSessions(bufferSize-1, currDBTime)
	for i := range sessions {
		sessionChan <- &sessions[i]
	}

	env := fixtures.Get(integrationtest.EnvironmentFixture.Key).(environment.Environment)
	currDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + currDBTime.Format(timeFormatString)

	waitTime := 10 * time.Second
	time.Sleep(waitTime)

	currDBCount, err := ssn.DB(currDBName).C(constants.ConnCollection).Count()
	require.Nil(t, err)
	require.Equal(t, 4, currDBCount)
	ssn.Close()

	close(sessionChan)
	for i := range errs {
		t.Fatal(errs[i])
	}
}

func TestMetaDBRecords(t *testing.T) {
	//If this test fails, its probably because of the bad waits needed
	//Ideally these would be replaced with a time-bounded check loop
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	ritaWriter := fixtures.GetWithSkip(t, streamingRITATimeIntervalWriterFixture.Key).(output.SessionWriter)

	clock := fixtures.Get(clockFixture.Key).(*clock.Mock)

	env := fixtures.Get(integrationtest.EnvironmentFixture.Key).(environment.Environment)

	//get the mongo session ready for checking
	mongoContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)
	ssn, err := mongoContainer.NewSession()
	if err != nil {
		t.Fatal(err)
	}

	sessionChan := make(chan *session.Aggregate, bufferSize)

	errChan := ritaWriter.Write(sessionChan)
	var errs []error
	go func() {
		for err := range errChan {
			errs = append(errs, err)
		}
	}()

	//We need to wait for asynchronous operations to finish at several
	//points in the test.
	waitTime := 10 * time.Second

	currDBTime := clock.Now().In(timezone)
	prevDBTime := currDBTime.Add(-1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	nextDBTime := currDBTime.Add(1 * time.Duration(intervalLengthMillis) * time.Millisecond)
	currDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + currDBTime.Format(timeFormatString)
	prevDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + prevDBTime.Format(timeFormatString)
	nextDBName := env.GetOutputConfig().GetRITAConfig().GetDBRoot() + "-" + nextDBTime.Format(timeFormatString)

	prevSessions := generateNSessions(bufferSize, prevDBTime)

	for i := range prevSessions {
		sessionChan <- &prevSessions[i]
	}

	time.Sleep(waitTime)

	dbInfo := rita.DBMetaInfo{}
	ssn.DB(env.GetOutputConfig().GetRITAConfig().GetMetaDB()).C(constants.MetaDBDatabasesCollection).Find(
		bson.M{"name": prevDBName},
	).One(&dbInfo)

	require.Equal(t, prevDBName, dbInfo.Name)
	require.Equal(t, false, dbInfo.ImportFinished)

	currSessions := generateNSessions(bufferSize, currDBTime)

	for i := range currSessions {
		sessionChan <- &currSessions[i]
	}

	time.Sleep(waitTime)

	dbInfo = rita.DBMetaInfo{}
	ssn.DB(env.GetOutputConfig().GetRITAConfig().GetMetaDB()).C(constants.MetaDBDatabasesCollection).Find(
		bson.M{"name": currDBName},
	).One(&dbInfo)

	require.Equal(t, currDBName, dbInfo.Name)
	require.Equal(t, false, dbInfo.ImportFinished)

	clock.Add(time.Duration(gracePeriodCutoffMillis) * time.Millisecond)

	time.Sleep(waitTime)

	dbInfo = rita.DBMetaInfo{}
	ssn.DB(env.GetOutputConfig().GetRITAConfig().GetMetaDB()).C(constants.MetaDBDatabasesCollection).Find(
		bson.M{"name": prevDBName},
	).One(&dbInfo)

	require.Equal(t, prevDBName, dbInfo.Name)
	require.Equal(t, true, dbInfo.ImportFinished)

	clock.Add(time.Duration(intervalLengthMillis-gracePeriodCutoffMillis) * time.Millisecond)

	time.Sleep(waitTime)

	nextSessions := generateNSessions(bufferSize, nextDBTime)
	for i := range nextSessions {
		sessionChan <- &nextSessions[i]
	}

	time.Sleep(waitTime)

	dbInfo = rita.DBMetaInfo{}
	ssn.DB(env.GetOutputConfig().GetRITAConfig().GetMetaDB()).C(constants.MetaDBDatabasesCollection).Find(
		bson.M{"name": nextDBName},
	).One(&dbInfo)

	require.Equal(t, nextDBName, dbInfo.Name)
	require.Equal(t, false, dbInfo.ImportFinished)

	clock.Add(time.Duration(gracePeriodCutoffMillis) * time.Millisecond)

	time.Sleep(waitTime)

	dbInfo = rita.DBMetaInfo{}
	ssn.DB(env.GetOutputConfig().GetRITAConfig().GetMetaDB()).C(constants.MetaDBDatabasesCollection).Find(
		bson.M{"name": currDBName},
	).One(&dbInfo)

	require.Equal(t, currDBName, dbInfo.Name)
	require.Equal(t, true, dbInfo.ImportFinished)

	ssn.Close()

	close(sessionChan)
	for i := range errs {
		t.Fatal(errs[i])
	}
}

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
