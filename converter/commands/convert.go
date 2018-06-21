package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/ipfix"
	input "github.com/activecm/ipfix-rita/converter/ipfix/mgologstash"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/partitioning"
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
	flowData, errors := reader.Drain(ctx)

	var numWorkers int32 = 5
	workerBuff := 5
	partitioner := partitioning.NewHashPartitioner(numWorkers, workerBuff)

	flowPartitions, errors2 := partitioner.Partition(ctx, flowData)

	exporterMap := stitching.NewExporterMap(numWorkers)
	writer := output.SpewRITAConnWriter{}

	//TODO: Abstract to StitcherPool and handle errors
	for id := range flowPartitions {
		go func(ctx context.Context, env environment.Environment,
			exporterMap stitching.ExporterMap, writer output.SessionWriter,
			partition <-chan ipfix.Flow, int id) {

			stitcher := stitching.NewStitcher(env, exporterMap, writer, id)
			for {
				select {
				case <-ctx.Done():
					return
				case flow <- partition:
					_ = stitcher.Stitch(flow)
				}
			}
		}(ctx, env, exporterMap, writer, flowPartitions[id], id)
	}

	_ = flowPartitions
	_ = errors2
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
