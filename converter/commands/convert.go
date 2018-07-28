package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/activecm/ipfix-rita/converter/environment"
	input "github.com/activecm/ipfix-rita/converter/input/mgologstash"
	"github.com/activecm/ipfix-rita/converter/logging"
	buffered "github.com/activecm/ipfix-rita/converter/output/rita/buffered/dates"
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

	//use CTRL-C as our signal to wrap up and exit
	ctx, _ := interruptContext(env.Logger)

	//TODO: Decide whether or not to expose these options
	//TODO: Decide on how to scale these options depending on the specs
	//of the computer

	//pollWait is how long to wait before checking if the input buffer has
	//more data
	pollWait := 30 * time.Second

	//inputBufferSize is how much data is stored in RAM at a time
	//for IDBulkBuffer this is also how much data is transferred in a single request
	inputBufferSize := 10000

	//Readers read from Buffers
	//reader will poll the MongoDB IDBulkBuffer which fetches records
	//in order of the ID field (usually insertion order)
	inputDB, err := input.NewLogstashMongoInputDB(
		env.GetInputConfig().GetLogstashMongoDBConfig(),
	)
	if err != nil {
		return err
	}
	reader := input.NewReader(
		input.NewIDBulkBuffer(
			inputDB.NewInputConnection(),
			inputBufferSize,
			env.Logger,
		),
		pollWait,
		env.Logger,
	)

	//sameSessionThreshold determines is used in the process of determining
	//whether two flows should be stitched together or not.
	//If the time between one flow ending and the other flow starting
	//exceeds sameSessionThreshold, they will not be stitched together.
	sameSessionThreshold := 1000 * 60 //milliseconds

	//how many stitching workers to use. The stitching workers
	//are assigned work by hash partitioning. Flows which may be stitched
	//together are guaranteed to handled by the same worker.
	numStitchers := 20

	//each woker has an input buffer.
	//if the inputBufferSize is evenly split among the stitching workers
	//then each stitcher needs a buffer at least as big as
	//inputBufferSize / numStitchers.
	stitcherBufferSize := inputBufferSize / numStitchers

	//matcherSize determines how many session.AggregateQuery
	//objects can be considered a candidate for matching (stitching)
	//at any given time
	//Increasing this value will likely increase the accuracy
	//of the results. However, a larger matcher likely takes
	//more resources (RAM/ CPU) to run at the same level of performance.
	matcherSize := 5000
	//when the matcher must flush connection records out,
	//the matcher will flush to matcherFlushToPercent * matcherSize
	matcherFlushToPercent := 0.9

	//outputBufferSize is used to set the size of the buffered channel
	//leading to the output.SessionWriter. It should be able to handle
	//at least as many records as in the input buffer.
	outputBufferSize := inputBufferSize
	//if more data could come out of the matcher via flushing
	//than the input buffer, use that to guide the output buffer size
	//Divide by 2 is a rough estimate of how many flows may be flushed at once
	if outputBufferSize < matcherSize/2 {
		outputBufferSize = matcherSize / 2
	}

	//the stitchingManager reads input from the input channel
	//and assigns the input flows to a pool stitcher workers.
	//Additionally, it manages the Matcher which is responsible
	//for providing the (CRUD+Flush) data structure needed for
	//stitching.
	stitchingManager := stitching.NewManager(
		int64(sameSessionThreshold),
		int32(numStitchers),
		stitcherBufferSize,
		outputBufferSize,
		int64(matcherSize),
		matcherFlushToPercent,
		env.Logger,
	)

	//flushDeadline determines how long data may sit in a buffer
	//before it is exported to MongoDB
	flushDeadline := 1 * time.Minute
	//bulkBatchSize is how much data is shipped to MongoDB at a time
	bulkBatchSize := outputBufferSize
	//NewBufferedRITAConnDateWriter creates a MongoDB/RITA conn-record writer
	//which splits output records up by the time the connection finished
	writer, err := buffered.NewBufferedRITAConnDateWriter(
		env.GetOutputConfig().GetRITAConfig(),
		env.GetIPFIXConfig(),
		bulkBatchSize, flushDeadline,
		env.Logger,
	)
	if err != nil {
		return err
	}

	//input channels
	inputData, inputErrors := reader.Drain(ctx)
	//run the stitching manager and get the output channels
	stitchingOutput, stitchingErrors := stitchingManager.RunAsync(inputData)
	//start the writer
	writingErrors := writer.Write(stitchingOutput)

	//Go-ism for waiting for several channels to close
	//process the errors on the main thread
	//the error channels will close when the component has exited
	for {
		select {
		case err, ok := <-inputErrors:
			if !ok {
				env.Info("input errors closed", nil)
				inputErrors = nil //nil channels cna't be selected
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
