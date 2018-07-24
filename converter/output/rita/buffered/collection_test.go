package buffered_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestCollection(t *testing.T) {
	testCollName := "TEST_BUFFERED_COLLECTION"
	integrationtest.RegisterDependenciesResetFunc(func(t *testing.T, deps *integrationtest.Dependencies) {
		coll := deps.Env.DB.NewCollection(testCollName)
		coll.DropCollection()
		coll.Database.Session.Close()
	})

	env := integrationtest.GetDependencies(t).Env
	defer integrationtest.CloseDependencies()

	coll := env.DB.NewCollection(testCollName)

	var bufferedColl buffered.Collection
	buffered.InitializeCollection(&bufferedColl, coll, 5)
	var inRecords []bson.M
	for i := 0; i < 11; i++ {
		inRecords = append(inRecords, bson.M{"test": i})
	}
	for i := range inRecords {
		err := bufferedColl.Insert(inRecords[i])
		require.Nil(t, err)
	}
	err := bufferedColl.Flush()
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
