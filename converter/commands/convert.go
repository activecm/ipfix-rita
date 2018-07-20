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
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/stitching"
	"github.com/urfave/cli"
	//	_ "net/http/pprof" //Profiling
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
	/*
		Profiling:
		go func() {
			fmt.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	*/

	env, err := environment.NewDefaultEnvironment()
	if err != nil {
		return err
	}

	ctx, cancel := interruptContext(env.Logger)
	defer cancel()

	pollWait := 30 * time.Second
	/*
		var readers []ipfix.Reader
		for i := 0; i < 1; i++ {
			readers = append(readers,
				input.NewReader(
					input.NewIDBuffer(
						env.DB.NewInputConnection(),
						env.Logger,
					),
					pollWait,
					env.Logger,
				),
			)
		}

		inputData, inputErrors := ipfix.DrainNReaders(readers, ctx)
	*/
	reader := input.NewReader(
		input.NewIDBulkBuffer(
			env.DB.NewInputConnection(),
			env.Logger,
		),
		pollWait,
		env.Logger,
	)
	inputData, inputErrors := reader.Drain(ctx)

	sameSessionThreshold := int64(1000 * 60) //milliseconds
	numStitchers := int32(20)
	stitcherBufferSize := 50
	outputBufferSize := int(numStitchers) * stitcherBufferSize
	sessionsCollMaxSize := 5000

	stitchingManager := stitching.NewManager(
		sameSessionThreshold,
		numStitchers,
		stitcherBufferSize,
		outputBufferSize,
		sessionsCollMaxSize,
		env.Logger,
	)

	stitchingOutput, stitchingErrors := stitchingManager.RunAsync(inputData, env.DB)

	//var writer output.SpewRITAConnWriter
	writer := output.RITAConnWriter{
		Environment: env,
	}
	writingErrors := writer.Write(stitchingOutput)

	for {
		select {
		case err, ok := <-inputErrors:
			if !ok {
				env.Info("input errors closed", nil)
				inputErrors = nil
				break
			}
			env.Error(err, logging.Fields{"component": "input"})
		case err, ok := <-stitchingErrors:
			if !ok {
				env.Info("stitching errors closed", nil)
				stitchingErrors = nil
				break
			}
			env.Error(err, logging.Fields{"component": "stitching"})
		case err, ok := <-writingErrors:
			if !ok {
				env.Info("output errors closed", nil)
				writingErrors = nil
				break
			}
			env.Error(err, logging.Fields{"component": "output"})
		}
		if inputErrors == nil && stitchingErrors == nil && writingErrors == nil {
			break
		}
	}
	env.Info("main thread exiting", nil)
	return nil
}

func interruptContext(log logging.Logger) (context.Context, func()) {
	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			log.Info("CTRL-C Received", nil)
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel /*func() { signal.Stop(sigChan); cancel() }*/
}
