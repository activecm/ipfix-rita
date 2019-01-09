package integrationtest_test

import (
	"os"
	"testing"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/integrationtest"
	"github.com/stretchr/testify/require"
)

var fixtureManager *integrationtest.FixtureManager

func TestMain(m *testing.M) {
	fixtureManager = integrationtest.NewFixtureManager()
	fixtureManager.RegisterFixture(integrationtest.EnvironmentFixture)
	fixtureManager.BeginTestPackage()
	code := m.Run()
	fixtureManager.EndTestPackage()
	os.Exit(code)
}

func TestEnvironmentFixture(t *testing.T) {
	fixtures := fixtureManager.BeginTest(t)
	envIface := fixtures.Get(integrationtest.EnvironmentFixture.Key)
	env, ok := envIface.(environment.Environment)
	require.True(t, ok)
	require.NotNil(t, env.GetFilteringConfig())
}

//TODO: Ensure each piece of the lifecycle works as expected
//TODO: Ensure writes to input map are inneffectual
