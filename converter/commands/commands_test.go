package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestRegisterCommand(t *testing.T) {
	registry := Registry{}
	newCommand := cli.Command{
		Name: "Command1",
	}
	registry.RegisterCommands(newCommand)
	registeredCommands := registry.GetCommands()
	require.Equal(t, newCommand, registeredCommands[0])
}

func TestRegisterMultipleCommands(t *testing.T) {
	registry := Registry{}
	newCommand0 := cli.Command{
		Name: "Command0",
	}
	newCommand1 := cli.Command{
		Name: "Command1",
	}
	registry.RegisterCommands(newCommand0, newCommand1)
	registeredCommands := registry.GetCommands()
	require.Equal(t, newCommand0, registeredCommands[0])
	require.Equal(t, newCommand1, registeredCommands[1])
}

func TestRegistrySingleton(t *testing.T) {
	r0 := GetRegistry()
	r1 := GetRegistry()
	require.NotNil(t, r0)
	require.Equal(t, r0, r1)
}
