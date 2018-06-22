package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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

	for {
		select {
		case err, ok := <-inputErrors:
			if !ok {
				fmt.Println("Input Errors Closed")
				inputErrors = nil
				break
			}
			fmt.Fprintf(os.Stderr, "%+v\n", err)
		case err, ok := <-stitchingErrors:
			if !ok {
				fmt.Println("Stitching Errors Closed")
				stitchingErrors = nil
				break
			}
			fmt.Fprintf(os.Stderr, "%+v\n", err)
		}
		if inputErrors == nil && stitchingErrors == nil {
			break
		}
	}
	fmt.Println("Main thread exiting")
	return nil
}

func interruptContext() (context.Context, func()) {
	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			fmt.Printf("\nRecieved CTRL-C\n")
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() { /*signal.Stop(sigChan);*/ cancel() }
}
