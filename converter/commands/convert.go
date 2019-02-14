package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/filter"
	input "github.com/activecm/ipfix-rita/converter/input/mgologstash"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/output"
	batchRITAOutput "github.com/activecm/ipfix-rita/converter/output/rita/batch/dates"
	streamingRITAOutput "github.com/activecm/ipfix-rita/converter/output/rita/streaming/dates"
	"github.com/activecm/ipfix-rita/converter/stitching"
	"github.com/benbjohnson/clock"
	"github.com/urfave/cli"
)

func init() {
	noRotateFlag := cli.BoolFlag{
		Name:  "no-rotate, r",
		Usage: "Do not create and rotate daily databases. Instead, split the incoming flows based on their timestamps into day-by-day databases and make them available to RITA when IPFIX-RITA shuts down.",
	}

	GetRegistry().RegisterCommands(cli.Command{
		Name:  "run",
		Usage: "Run the IPFIX-RITA converter",
		Flags: []cli.Flag{noRotateFlag},
		Action: func(c *cli.Context) error {
			env, err := environment.NewDefaultEnvironment()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			noRotate := c.Bool("no-rotate")
			err = convert(env, noRotate)
			if err != nil {
				env.Logger.Error(err, nil)
				return cli.NewExitError(nil, 1)
			}
			return nil
		},
	})
}

func convert(env environment.Environment, noRotate bool) error {

	//use CTRL-C as our signal to wrap up and exit
	ctx, _ := interruptContext(env.Logger)

	//TODO: Decide whether or not to expose these options
	//TODO: Decide on how to scale these options depending on the specs
	//of the computer

	//-------------------------------Input setup-------------------------------

	//pollWait is how long to wait before checking if the input buffer has
	//more data
	pollWait := 30 * time.Second

	//inputBufferSize is how much data is stored in RAM at a time
	//for IDBulkBuffer this is also how much data is transferred in a single request
	inputBufferSize := int64(10000)

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

	//-------------------------------Filter setup-------------------------------

	//Create the filter which will filter out flows as specified by the
	//Filter config section
	internalNets, errs := env.GetFilteringConfig().GetInternalSubnets()
	if len(errs) != 0 {
		for _, err := range errs {
			env.Logger.Error(err, nil)
		}
		return errors.New("unable to parse filtering config")
	}
	neverIncludeNets, errs := env.GetFilteringConfig().GetNeverIncludeSubnets()
	if len(errs) != 0 {
		for _, err := range errs {
			env.Logger.Error(err, nil)
		}
		return errors.New("unable to parse filtering config")
	}
	alwaysIncludeNets, errs := env.GetFilteringConfig().GetAlwaysIncludeSubnets()
	if len(errs) != 0 {
		for _, err := range errs {
			env.Logger.Error(err, nil)
		}
		return errors.New("unable to parse filtering config")
	}

	flowFilter := filter.NewFlowBlacklist(
		internalNets,
		neverIncludeNets,
		alwaysIncludeNets,
	)

	//------------------------------Stitching setup------------------------------

	//sameSessionThreshold determines is used in the process of determining
	//whether two flows should be stitched together or not.
	//If the time between one flow ending and the other flow starting
	//exceeds sameSessionThreshold, they will not be stitched together.
	sameSessionThreshold := int64(1000 * 60) //milliseconds

	//how many stitching workers to use. The stitching workers
	//are assigned work by hash partitioning. Flows which may be stitched
	//together are guaranteed to handled by the same worker.
	numStitchers := int32(20)

	//each woker has an input buffer.
	//if the inputBufferSize is evenly split among the stitching workers
	//then each stitcher needs a buffer at least as big as
	//inputBufferSize / numStitchers.
	stitcherBufferSize := inputBufferSize / int64(numStitchers)

	//matcherSize determines how many session.AggregateQuery
	//objects can be considered a candidate for matching (stitching)
	//at any given time
	//Increasing this value will likely increase the accuracy
	//of the results. However, a larger matcher likely takes
	//more resources (RAM/ CPU) to run at the same level of performance.
	matcherSize := int64(5000)
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
		sameSessionThreshold,
		numStitchers,
		stitcherBufferSize,
		outputBufferSize,
		matcherSize,
		matcherFlushToPercent,
		flowFilter,
		env.Logger,
	)

	//-------------------------------Output setup-------------------------------

	//flushDeadline determines how long data may sit in a buffer
	//before it is exported to MongoDB
	flushDeadline := 1 * time.Minute
	//bulkBatchSize is how much data is shipped to MongoDB at a time
	bulkBatchSize := outputBufferSize

	var writer output.SessionWriter

	if !noRotate {
		dayRotationPeriodMillis := int64(1000 * 60 * 60 * 24) //daily datasets
		gracePeriodMillis := int64(1000 * 60 * 5)             //analysis can happen after 12:05 am
		dateFormatString := "2006-01-02"

		//NewStreamingRITATimeIntervalWriter creates a MongoDB/RITA conn-record writer
		//which splits output records up based on the time the connection finished
		writer, err = streamingRITAOutput.NewStreamingRITATimeIntervalWriter(
			env.GetOutputConfig().GetRITAConfig(),
			internalNets,
			bulkBatchSize, flushDeadline,
			dayRotationPeriodMillis, gracePeriodMillis,
			clock.New(), time.Local, dateFormatString,
			env.Logger,
		)
		if err != nil {
			return err
		}
	} else {
		writer, err = batchRITAOutput.NewBatchRITAConnDateWriter(
			env.GetOutputConfig().GetRITAConfig(),
			internalNets,
			bulkBatchSize, flushDeadline,
			env.Logger,
		)
		if err != nil {
			return err
		}
		env.Info("Database rotation has been disabled", nil)
	}

	//-------------------------------Execution-------------------------------

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
			logTime := time.Now().Format("2006-01-02T15:04:05Z07:00")
			log.Info(logTime+" CTRL-C Received", nil)
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel /*func() { signal.Stop(sigChan); cancel() }*/
}
