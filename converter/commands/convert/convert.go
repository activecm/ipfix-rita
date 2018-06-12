package convert

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/activecm/ipfix-rita/converter/commands"
	"github.com/activecm/ipfix-rita/converter/environment"
	input "github.com/activecm/ipfix-rita/converter/types/ipfix/mgologstash"
	"github.com/urfave/cli"
)

func init() {
	commands.GetRegistry().RegisterCommands(cli.Command{
		Name:  "run",
		Usage: "Run the IPFIX-RITA converter",
		Action: func(c *cli.Context) error {
			err := convert()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			return nil
		},
	})
}

func convert() error {
	env, err := environment.NewDefaultEnvironment()
	if err != nil {
		return err
	}
	reader := input.NewReader(input.NewIDBuffer(env.DB.NewInputConnection()), 30*time.Second)
	ctx, cancel := interruptContext()
	defer cancel()
	data, errors := reader.Drain(ctx)
	_ = data
	_ = errors
	return nil
}

func interruptContext() (context.Context, func()) {
	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() { signal.Stop(sigChan); cancel() }
}
