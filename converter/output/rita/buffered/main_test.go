package buffered_test

import (
	"os"
	"testing"

	"github.com/activecm/ipfix-rita/converter/integrationtest"
)

const testCollectionName = "BUFFERED_TEST_COLLECTION"

//TestMain is responsible for setting up and tearing down any
//resources needed by all tests
func TestMain(m *testing.M) {
	integrationtest.RegisterDependenciesResetFunc(func(t *testing.T, deps *integrationtest.Dependencies) {
		coll := deps.Env.DB.NewHelperCollection(testCollectionName)
		count, err := coll.Count()
		if count != 0 && err == nil {
			coll.DropCollection()
		}
		coll.Database.Session.Close()
	})
	returnCode := m.Run()
	integrationtest.CloseDependencies() //no effect if no integration tests run
	os.Exit(returnCode)
}
