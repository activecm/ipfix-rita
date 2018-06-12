package environmenttest_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/environmenttest"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTestWrapper(t *testing.T) {
	env, cleanup := environmenttest.SetupIntegrationTest(t)
	defer cleanup()
	require.NotNil(t, env.Config)
	require.NotNil(t, env.Logger)
	require.NotNil(t, env.DB.NewInputConnection())
}
