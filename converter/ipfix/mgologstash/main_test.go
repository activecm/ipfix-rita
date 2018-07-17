package mgologstash_test

import (
	"os"
	"testing"

	"github.com/activecm/ipfix-rita/converter/integrationtest"
)

//TestMain is responsible for setting up and tearing down any
//resources needed by all tests
func TestMain(m *testing.M) {
	returnCode := m.Run()
	integrationtest.CloseDependencies() //no effect if no integration tests run
	os.Exit(returnCode)
}
