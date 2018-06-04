package main

import (
	"os"

	"github.com/activecm/ipfix-rita/converter/commands"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "ipfix-rita"
	app.Version = "v0.0.0"
	app.Commands = commands.GetRegistry().GetCommands()

	app.Run(os.Args)
}
