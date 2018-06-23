package main

import (
	"os"
	"runtime"

	"github.com/activecm/ipfix-rita/converter/commands"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "ipfix-rita"
	app.Version = "v0.0.0"
	app.Commands = commands.GetRegistry().GetCommands()

	runtime.GOMAXPROCS(runtime.NumCPU())
	app.Run(os.Args)
}
