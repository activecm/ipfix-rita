package commands

import (
	"fmt"

	"github.com/urfave/cli"
)

func init() {
	GetRegistry().RegisterCommands(cli.Command{
		Name:  "run",
		Usage: "Run the IPFIX-RITA converter",
		Action: func(c *cli.Context) error {
			fmt.Println("Hello World")
			return nil
		},
	})
}
