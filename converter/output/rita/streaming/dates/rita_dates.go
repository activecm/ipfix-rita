package dates

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/output/rita"
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
)

type streamingRITATimeIntervalWriter struct {
	ritaDBManager           rita.OutputDB
	localNets               []net.IPNet
	collectionBufferSize    int64
	autoflushDeadline       time.Duration
	segmentTSFactory        SegmentRelativeTimestampFactory
	gracePeriodCutoffMillis int64
	timeFormatString        string

	inGracePeriod      bool
	currentSegmentTS   SegmentRelativeTimestamp
	previousCollection *buffered.AutoFlushCollection
	currentCollection  *buffered.AutoFlushCollection
}

func NewStreamingRITATimeIntervalWriter(ritaConf config.RITA, ipfixConf config.IPFIX,
	bufferSize int64, autoFlushTime time.Duration, intervalLengthMillis int64,
	gracePeriodCutoffMillis int64, timeFormatString string,
	log logging.Logger) (output.SessionWriter, error) {

	db, err := rita.NewOutputDB(ritaConf)
	if err != nil {
		return nil, errors.Wrap(err, "could not connecto to RITA MongoDB")
	}

	//parse local networks
	localNets, localNetsErrs := ipfixConf.GetLocalNetworks()
	if len(localNetsErrs) != 0 {
		for i := range localNetsErrs {
			log.Warn("could not parse local network", logging.Fields{"err": localNetsErrs[i]})
		}
	}

	return &streamingRITATimeIntervalWriter{
		ritaDBManager:           db,
		localNets:               localNets,
		collectionBufferSize:    bufferSize,
		autoflushDeadline:       autoFlushTime,
		segmentTSFactory:        SegmentRelativeTimestampFactory{segmentDurationMillis: intervalLengthMillis},
		gracePeriodCutoffMillis: gracePeriodCutoffMillis,
		timeFormatString:        timeFormatString,
	}, nil
}

func (s *streamingRITATimeIntervalWriter) newAutoFlushCollection(time time.Time,
	autoFlushErrChan chan<- error) (*buffered.AutoFlushCollection, error) {

	newColl, err := s.ritaDBManager.NewRITAOutputConnection(time.Format(s.timeFormatString))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start auto flusher for collection %s.%s", s.currentCollection.Database(), s.currentCollection.Name())
	}
	err = s.ritaDBManager.EnsureMetaDBRecordExists(newColl.Database.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start auto flusher for collection %s.%s", s.currentCollection.Database(), s.currentCollection.Name())
	}

	newAutoFlushCollection := buffered.NewAutoFlushCollection(newColl, s.collectionBufferSize, s.autoflushDeadline, autoFlushErrChan)
	started := s.currentCollection.StartAutoFlush()
	if !started {
		errmsg := fmt.Sprintf("failed to start auto flusher for collection %s.%s", s.currentCollection.Database(), s.currentCollection.Name())
		return nil, errors.New(errmsg)
	}
	return newAutoFlushCollection, nil
}

func (s *streamingRITATimeIntervalWriter) initializeCurrentSegmentAndGracePeriod(autoFlushErrChan chan<- error) error {
	currTime := time.Now()
	currTimeMillis := currTime.UnixNano() / 1000000

	s.currentSegmentTS = s.segmentTSFactory.GetSegmentRelativeTimestamp(currTimeMillis)
	s.inGracePeriod = s.currentSegmentTS.OffsetFromSegmentStartMillis < s.gracePeriodCutoffMillis

	//set previousCollection if needed
	if s.inGracePeriod {
		//time.Unix(seconds, nanoseconds)
		//1000 milliseconds per second, 1000 nanosecodns to a microsecond. 1000 microseconds to a millisecond
		prevTimeMillis := s.currentSegmentTS.SegmentStartMillis - s.currentSegmentTS.SegmentDurationMillis
		prevTime := time.Unix(prevTimeMillis/1000, (prevTimeMillis%1000)*1000*1000)

		var err error
		s.previousCollection, err = s.newAutoFlushCollection(prevTime, autoFlushErrChan)
		if err != nil {
			return errors.Wrap(err, "could not initialize streaming RITA interval writer")
		}
	}

	//set currentCollection
	var err error
	s.currentCollection, err = s.newAutoFlushCollection(currTime, autoFlushErrChan)
	if err != nil {
		return errors.Wrap(err, "could not initialize streaming RITA interval writer")
	}
	return nil
}

func (s *streamingRITATimeIntervalWriter) at(unixTSMillis int64) <-chan time.Time {
	currTime := time.Now()
	currTimeMillis := currTime.UnixNano() / 1000000
	if currTimeMillis >= unixTSMillis {
		instantChan := make(chan time.Time, 1)
		instantChan <- currTime
		return instantChan
	}
	return time.After(
		time.Duration(unixTSMillis-currTimeMillis) * time.Millisecond,
	)
}

func (s *streamingRITATimeIntervalWriter) getNextUpdateChan() <-chan time.Time {
	if s.inGracePeriod {
		//update after the grace period expires
		return s.at(s.currentSegmentTS.SegmentStartMillis + s.gracePeriodCutoffMillis)
	}
	//update after the time segment changes
	return s.at(s.currentSegmentTS.SegmentStartMillis + s.currentSegmentTS.SegmentDurationMillis)
}

func (s *streamingRITATimeIntervalWriter) flushLoop(breakOnErrorContext context.Context,
	cancelOnError func(), wg *sync.WaitGroup, errsOut chan<- error) {

	updateChan := s.getNextUpdateChan()

FlushLoop:
	for {
		select {
		case <-breakOnErrorContext.Done():
			break FlushLoop
		case <-updateChan:

			//trash the time sent on the updateChan since the scheduler might
			//have been lazy and might have blocked us from getting to it instantly
			currTime := time.Now()
			currTimeMillis := currTime.UnixNano() / 1000000

			s.currentSegmentTS = s.segmentTSFactory.GetSegmentRelativeTimestamp(currTimeMillis)
			s.inGracePeriod = s.currentSegmentTS.OffsetFromSegmentStartMillis < s.gracePeriodCutoffMillis

			//assert that s.inGracePeriod toggled
			if s.inGracePeriod {
				//Beginning of grace period, different time segment

				//set previousCollection
				s.previousCollection = s.currentCollection

				var err error
				//set currentCollection
				s.currentCollection, err = s.newAutoFlushCollection(currTime, errsOut)
				if err != nil {
					errsOut <- err
					break FlushLoop
				}
			} else {
				//End of grace period, same segment

				//Flush the previous collection and close it out
				s.previousCollection.Flush()
				s.previousCollection.Close()
				err := s.ritaDBManager.MarkImportFinishedInMetaDB(s.previousCollection.Database())
				if err != nil {
					errsOut <- err
					break FlushLoop
				}
				s.previousCollection = nil

			}

			//Note that getNextUpdateChan doesn't use the cached timestamp
			//in currentSegmentTS since it may take a decent amount of time
			//to flush a buffer to MongoDB
			updateChan = s.getNextUpdateChan()
		}
	}

	//wrap up the previous collection if it is open
	if s.previousCollection != nil {
		s.previousCollection.Flush()
		s.previousCollection.Close()
		err := s.ritaDBManager.MarkImportFinishedInMetaDB(s.previousCollection.Database())
		if err != nil {
			errsOut <- err
		}
	}

	//Wrap up the current collection
	s.currentCollection.Flush()
	s.currentCollection.Close()
	err := s.ritaDBManager.MarkImportFinishedInMetaDB(s.currentCollection.Database())
	if err != nil {
		errsOut <- err
	}

	wg.Done()
	cancelOnError()
}

func (s *streamingRITATimeIntervalWriter) writeLoop(breakOnErrorContext context.Context,
	cancelOnError func(), wg *sync.WaitGroup, sessions <-chan *session.Aggregate, errsOut chan<- error) {

WriteLoop:
	for {
		select {
		case <-breakOnErrorContext.Done():
			break WriteLoop
		case sess, ok := <-sessions:
			if !ok {
				break WriteLoop
			}

			sessEndMillis := sess.FlowEndMilliseconds()
			sessEndSegmentTS := s.segmentTSFactory.GetSegmentRelativeTimestamp(sessEndMillis)

			//TODO: Lock over currentSegmentTS, inGracePeriod, currentCollection, previousCollection

			//we drop the sameDuration check off the result from the next call
			//since we know we are only using a single segmentTSFactory
			segOffset, _ := s.currentSegmentTS.SegmentOffsetFrom(sessEndSegmentTS)

			if segOffset == 0 {
				var ritaConn parsetypes.Conn
				sess.ToRITAConn(&ritaConn, s.isIPLocal)

				//Insert into today's db
				s.currentCollection.Insert(ritaConn)

			} else if segOffset == -1 && s.inGracePeriod {
				var ritaConn parsetypes.Conn
				sess.ToRITAConn(&ritaConn, s.isIPLocal)

				//Insert into yesterday's db
				s.previousCollection.Insert(ritaConn)
			} else {
				//Drop the connection record
			}
		}
	}

	wg.Done()
	cancelOnError()
}

func (s *streamingRITATimeIntervalWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	//initialize the current time derived variables (including MongoDB databases)
	errs := make(chan error)

	err := s.initializeCurrentSegmentAndGracePeriod(errs)
	if err != nil {
		//If we couldn't contact MongoDB, we need to return an error
		//errs has a buffer for an error so we don't deadlock ourselves here
		errs = make(chan error, 1)
		errs <- err
		close(errs)
		return errs
	}

	//start the flush to maintain the previous and current databases
	go func() {
		wg := new(sync.WaitGroup)
		breakOnErrorContext, cancelOnError := context.WithCancel(context.Background())
		go s.flushLoop(breakOnErrorContext, cancelOnError, wg, errs)
		go s.writeLoop(breakOnErrorContext, cancelOnError, wg, sessions, errs)
		wg.Wait()
		close(errs)
	}()
	return errs
}

func (s *streamingRITATimeIntervalWriter) isIPLocal(ipAddrStr string) bool {
	ipAddr := net.ParseIP(ipAddrStr)
	for i := range s.localNets {
		if s.localNets[i].Contains(ipAddr) {
			return true
		}
	}
	return false
}
