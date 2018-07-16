package integrationtest_test

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/stretchr/testify/require"
)

func TestInitDependencies(t *testing.T) {
	integrationDeps := integrationtest.GetDependencies(t)
	require.NotNil(t, integrationDeps)
	env := integrationDeps.GetFreshEnvironment(t)
	require.NotNil(t, env.Config)
	require.NotNil(t, env.Logger)
	require.NotNil(t, env.DB.NewInputConnection())
	integrationtest.CloseDependencies()
}
