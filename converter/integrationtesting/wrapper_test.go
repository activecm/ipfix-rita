package integrationtesting_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/integrationtesting"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTestWrapper(t *testing.T) {
	env, cleanup := integrationtesting.SetupIntegrationTest(t)
	require.NotNil(t, env.Config)
	require.NotNil(t, env.Logger)
	require.NotNil(t, env.DB.Session)
	defer cleanup()
}
