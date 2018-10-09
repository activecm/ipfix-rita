package buffered_test

import (
	"testing"
	"time"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/stretchr/testify/require"
	"github.com/globalsign/mgo/bson"
)

func TestAutoFlushCollectionBufferedWrites(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	env := fixtures.GetWithSkip(t, integrationtest.EnvironmentFixture.Key).(environment.Environment)
	mongoDBContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)
	ssn, err := mongoDBContainer.NewSession()
	if err != nil {
		env.Error(err, nil)
		t.FailNow()
	}
	coll := ssn.DB(testDBName).C(testCollectionName)

	errs := make(chan error, 100)
	autoFlushColl := buffered.NewAutoFlushCollection(coll, 5, 5*time.Second, errs)
	autoFlushColl.StartAutoFlush()
	var inRecords []bson.M
	for i := 0; i < 11; i++ {
		inRecords = append(inRecords, bson.M{"test": i})
	}
	for i := range inRecords {
		autoFlushColl.Insert(inRecords[i])
	}

	autoFlushColl.Flush()

	var outRecords []bson.M
	err = coll.Find(nil).All(&outRecords)
	require.Nil(t, err)
	require.Len(t, outRecords, len(inRecords))
	for i := range inRecords {
		found := false
		for j := range outRecords {
			if inRecords[i]["test"] == outRecords[j]["test"] {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Did not find element %+v", inRecords[i])
		}
		require.True(t, found)
	}

	autoFlushColl.Close()
	close(errs)

	err, ok := <-errs
	if ok {
		env.Error(err, nil)
		for err = range errs {
			env.Error(err, nil)
		}
		t.FailNow()
	}
}

func TestAutoFlushCollectionAutoFlush(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	env := fixtures.GetWithSkip(t, integrationtest.EnvironmentFixture.Key).(environment.Environment)
	mongoDBContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)
	ssn, err := mongoDBContainer.NewSession()
	if err != nil {
		env.Error(err, nil)
		t.FailNow()
	}
	coll := ssn.DB(testDBName).C(testCollectionName)

	errs := make(chan error, 100)
	buffSize := 5
	deadlineInterval := 1 * time.Second
	autoFlushColl := buffered.NewAutoFlushCollection(coll, 5, 1*time.Second, errs)
	autoFlushColl.StartAutoFlush()
	var inRecords []bson.M
	for i := 0; i < buffSize-1; i++ {
		inRecords = append(inRecords, bson.M{"test": i})
	}
	for i := range inRecords {
		autoFlushColl.Insert(inRecords[i])
	}
	//wait for the auto flush to happen, give some time for the sheduler
	//to run the goroutine
	time.Sleep(deadlineInterval + 10*time.Millisecond)

	//notice no flush

	var outRecords []bson.M
	err = coll.Find(nil).All(&outRecords)
	require.Nil(t, err)
	require.Len(t, outRecords, len(inRecords))
	for i := range inRecords {
		found := false
		for j := range outRecords {
			if inRecords[i]["test"] == outRecords[j]["test"] {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Did not find element %+v", inRecords[i])
		}
		require.True(t, found)
	}

	autoFlushColl.Close()
	close(errs)
	err, ok := <-errs
	if ok {
		env.Error(err, nil)
		for err = range errs {
			env.Error(err, nil)
		}
		t.FailNow()
	}

}
