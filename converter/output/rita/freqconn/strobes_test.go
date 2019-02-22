package freqconn_test

import (
	"fmt"
	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/activecm/ipfix-rita/converter/output/rita/constants"
	"github.com/activecm/ipfix-rita/converter/output/rita/freqconn"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//TestLoadFreqConnCollection loads the freqConn collection up with data and pulls it down
//with the LoadFreqConnCollection function.
func TestLoadFreqConnCollection(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	mongoDBContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)

	ssn, err := mongoDBContainer.NewSession()
	require.Nil(t, err, "Could not connect to MongoDB")
	defer ssn.Close()

	testDB := ssn.DB(testDBName)

	// Populate the collection
	for i := 0; i < 100; i++ {
		err = testDB.C(constants.StrobesCollection).Insert(&freqconn.FreqConn{
			UConnPair: freqconn.UConnPair{
				Src: fmt.Sprintf("%d.%d.%d.%d", i, i, i, i),
				Dst: fmt.Sprintf("%d.%d.%d.%d", i+1, i+1, i+1, i+1),
			},
			ConnectionCount: i,
		})
		require.Nil(t, err, "Could not insert test data")
	}

	freqConnNotifier := freqconn.NewStrobesNotifier(testDB, nil)

	// Try to read the data
	data, err := freqConnNotifier.LoadFreqConnCollection()
	require.Nil(t, err, "Could not fetch freqconn entries")

	require.Len(t, data, 100, "Number of retrieved records does not match inserted data")

	for connPair, count := range data {
		srcStr := fmt.Sprintf("%d.%d.%d.%d", count, count, count, count)
		destStr := fmt.Sprintf("%d.%d.%d.%d", count+1, count+1, count+1, count+1)
		require.Equal(t, srcStr, connPair.Src, "Data retrieved does not match the original data")
		require.Equal(t, destStr, connPair.Dst, "Data retrieved does not match the original data")
	}
}

//TestStrobesThresholdMet ensures ThresholdMet clears out any matching records in the conn
//collection and inserts a new record into the freqConn collection.
func TestStrobesThresholdMet(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	mongoDBContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)

	ssn, err := mongoDBContainer.NewSession()
	require.Nil(t, err, "Could not connect to MongoDB")
	defer ssn.Close()

	testDB := ssn.DB(testDBName)

	srcIP := "1.1.1.1"
	dstIP := "2.2.2.2"

	s := parsetypes.Conn{
		Source:      srcIP,
		Destination: dstIP,
	}

	for i := 0; i < testThreshold-1; i++ {
		err = testDB.C(constants.ConnCollection).Insert(&s)
		require.Nil(t, err, "Could not insert test data")
	}

	freqConnNotifier := freqconn.NewStrobesNotifier(testDB, nil)

	err = freqConnNotifier.ThresholdMet(freqconn.UConnPair{
		Src: srcIP,
		Dst: dstIP,
	}, testThreshold)

	require.Nil(t, err, "Could not delete existing conn records or create a new freqConn record")

	connCount, err := testDB.C(constants.ConnCollection).Count()
	require.Nil(t, err, "Could not count how many records remain in conn collection")
	require.Zero(t, connCount, "Matching records were not removed from the conn collection after ThresholdMet was ran")

	freqCount, err := testDB.C(constants.StrobesCollection).Count()
	require.Nil(t, err, "Could not count how many records exist in freqConn collection")
	require.Equal(t, 1, freqCount, "ThresholdMet did not create a single record in freqConn")

	var freqResult freqconn.FreqConn
	err = testDB.C(constants.StrobesCollection).Find(nil).One(&freqResult)
	require.Nil(t, err, "Could not check freqConn for new records after ThresholdMet was ran")

	require.Equal(t, srcIP, freqResult.Src, "Source IP in freqConn does not match the original address")
	require.Equal(t, dstIP, freqResult.Dst, "Destination IP in freqConn does not match the original address")
	require.Equal(t, testThreshold, freqResult.ConnectionCount, "Connection count in freqConn does not match the count passed to ThresholdMet")
}

//TestStrobesThresholdExceeded ensures ThresholdExceeded increments the connection_count
//for a given UConnPair in the freqConn collection.
func TestStrobesThresholdExceeded(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	mongoDBContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)

	ssn, err := mongoDBContainer.NewSession()
	require.Nil(t, err, "Could not connect to MongoDB")
	defer ssn.Close()

	testDB := ssn.DB(testDBName)

	uconn := freqconn.UConnPair{
		Src: "1.1.1.1",
		Dst: "2.2.2.2",
	}

	err = testDB.C(constants.StrobesCollection).Insert(&freqconn.FreqConn{
		UConnPair:       uconn,
		ConnectionCount: testThreshold,
	})
	require.Nil(t, err, "Could not populate freqConn with test data")

	freqNotifier := freqconn.NewStrobesNotifier(testDB, nil)

	incAmount := 10

	for i := testThreshold + 1; i <= testThreshold+incAmount; i++ {
		freqNotifier.ThresholdExceeded(uconn, i)
	}

	var freqResult freqconn.FreqConn
	err = testDB.C(constants.StrobesCollection).Find(&uconn).One(&freqResult)
	require.Nil(t, err, "Could not check freqConn for new records after ThresholdExceeded was run")

	require.Equal(t, testThreshold+incAmount, freqResult.ConnectionCount, "Connection count incorrect after calling ThresholdExceeded")
	require.Equal(t, uconn.Src, freqResult.Src)
	require.Equal(t, uconn.Dst, freqResult.Dst)
}

func TestStrobesWithAutoFlushCollection(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	mongoDBContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)

	ssn, err := mongoDBContainer.NewSession()
	require.Nil(t, err, "Could not connect to MongoDB")
	defer ssn.Close()

	testDB := ssn.DB(testDBName)

	srcIP := "1.1.1.1"
	dstIP := "2.2.2.2"

	s := parsetypes.Conn{
		Source:      srcIP,
		Destination: dstIP,
	}
	bufferSize := 200
	flushTime := 2 * time.Second

	//Load an autoflush collection up with data (but not to the point where it will flush because the buffer is full)
	autoFlushColl := buffered.NewAutoFlushCollection(testDB.C(constants.ConnCollection), int64(bufferSize), flushTime)

	errs := make(chan error, bufferSize)
	autoFlushColl.StartAutoFlush(errs, func() { t.FailNow() })

	for i := 0; i < bufferSize/2; i++ {
		err = autoFlushColl.Insert(&s)
		require.Nil(t, err, "Could not insert test data")
	}

	//Run the ThresholdMet method. This should flush the auto flush collection.
	//If it doesn't the collection will have records in it after we call ThresholdMet
	//and wait for the deadline to pass
	freqConnNotifier := freqconn.NewStrobesNotifier(testDB, autoFlushColl)

	err = freqConnNotifier.ThresholdMet(freqconn.UConnPair{
		Src: srcIP,
		Dst: dstIP,
	}, testThreshold)

	require.Nil(t, err, "Could not delete existing conn records or create a new freqConn record")

	//Wait for the auto flush deadline to pass
	time.Sleep(flushTime)

	connCount, err := testDB.C(constants.ConnCollection).Count()
	require.Nil(t, err, "Could not count how many records remain in conn collection")
	require.Zero(t, connCount, "Matching records were not removed from the conn collection/ the auto flush buffer after ThresholdMet was ran")

	freqCount, err := testDB.C(constants.StrobesCollection).Count()
	require.Nil(t, err, "Could not count how many records exist in freqConn collection")
	require.Equal(t, 1, freqCount, "ThresholdMet did not create a single record in freqConn")

	var freqResult freqconn.FreqConn
	err = testDB.C(constants.StrobesCollection).Find(nil).One(&freqResult)
	require.Nil(t, err, "Could not check freqConn for new records after ThresholdMet was ran")

	require.Equal(t, srcIP, freqResult.Src, "Source IP in freqConn does not match the original address")
	require.Equal(t, dstIP, freqResult.Dst, "Destination IP in freqConn does not match the original address")
	require.Equal(t, testThreshold, freqResult.ConnectionCount, "Connection count in freqConn does not match the count passed to ThresholdMet")
}
