package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/activecm/ipfix-rita/converter/environment"
	input "github.com/activecm/ipfix-rita/converter/ipfix/mgologstash"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/stitching"
	"github.com/urfave/cli"
)

func init() {
	GetRegistry().RegisterCommands(cli.Command{
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
	pollWait := 30 * time.Second
	reader := input.NewReader(input.NewIDBuffer(env.DB.NewInputConnection()), pollWait)
	ctx, cancel := interruptContext()
	defer cancel()
	flowData, inputErrors := reader.Drain(ctx)

	sameSessionThreshold := uint64(1000 * 60 * 60) //milliseconds
	var numStitchers int32 = 5
	stitcherBufferSize := 5

	var writer output.SpewRITAConnWriter

	stitchingManager := stitching.NewManager(sameSessionThreshold, stitcherBufferSize, numStitchers)

	stitchingErrors := stitchingManager.RunAsync(flowData, env.DB, writer)

	//Print errors to stderr while running
	channelsDone := 0

errorLoop:
	for {
		select {
		case err, ok := <-inputErrors:
			if !ok {
				channelsDone++
				if channelsDone == 2 {
					break errorLoop
				}
				break
			}
			fmt.Fprintf(os.Stderr, "%+v\n", err)
		case err, ok := <-stitchingErrors:
			if !ok {
				channelsDone++
				if channelsDone == 2 {
					break errorLoop
				}
				break
			}
			fmt.Fprintf(os.Stderr, "%+v\n", err)
		}
	}
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
