package buffered_test

import (
	"testing"

	"github.com/activecm/dbtest"
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/require"
)

func TestCollection(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	defer fixtureManager.EndTest(t)

	mongoDBContainer := fixtures.GetWithSkip(t, mongoContainerFixtureKey).(dbtest.MongoDBContainer)
	ssn, err := mongoDBContainer.NewSession()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	coll := ssn.DB(testDBName).C(testCollectionName)

	var bufferedColl buffered.Collection
	buffered.InitializeCollection(&bufferedColl, coll, 5)
	var inRecords []bson.M
	for i := 0; i < 11; i++ {
		inRecords = append(inRecords, bson.M{"test": i})
	}
	for i := range inRecords {
		err = bufferedColl.Insert(inRecords[i])
		require.Nil(t, err)
	}
	err = bufferedColl.Flush()
	require.Nil(t, err)

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

	err = bufferedColl.Close()
	require.Nil(t, err)
}
