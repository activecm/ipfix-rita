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
	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
)

type streamingRITATimeIntervalWriter struct {
	ritaDBManager           rita.RITADBManager
	localNets               []net.IPNet
	collectionBufferSize    int64
	autoflushDeadline       time.Duration
	segmentTSFactory        SegmentRelativeTimestampFactory
	gracePeriodCutoffMillis int64
	timeFormatString        string
	timezone                *time.Location

	clock              clock.Clock
	inGracePeriod      bool
	currentSegmentTS   SegmentRelativeTimestamp
	previousCollection *buffered.AutoFlushCollection
	currentCollection  *buffered.AutoFlushCollection
	collectionMutex    *sync.Mutex

	log logging.Logger
}

//NewStreamingRITATimeIntervalWriter creates a new session-> rita Conn
//writer which writes sessions to a different databases depending on which
//interval of time they fall in. Sessions with a FlowEndMilliseconds
//timestamp in the current time interval are always written
//to the current interval database. For a portion of the current interval,
//the grace period, sessions with a FlowEndMilliseconds timestamp in
//the previous time interval are written to the previous interval database.
//The streamingRITATimeIntervalWriter automatically buffers and flushes
//sessions as needed. Additionally, it automatically maintains the
//Metadatabase records for each collection. A database is marked as
//ImportFinished in the Metadatabase when the database is the previous
//database and the grace period expires.
func NewStreamingRITATimeIntervalWriter(ritaConf config.RITA, localNets []net.IPNet,
	bufferSize int64, autoFlushTime time.Duration, intervalLengthMillis int64,
	gracePeriodCutoffMillis int64, clock clock.Clock, timezone *time.Location, timeFormatString string,
	log logging.Logger) (output.SessionWriter, error) {

	db, err := rita.NewRITADBManager(ritaConf)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to RITA MongoDB")
	}

	return &streamingRITATimeIntervalWriter{
		ritaDBManager:        db,
		localNets:            localNets,
		collectionBufferSize: bufferSize,
		autoflushDeadline:    autoFlushTime,
		segmentTSFactory:     NewSegmentRelativeTimestampFactory(intervalLengthMillis, timezone),
		timezone:             timezone,
		clock:                clock,
		gracePeriodCutoffMillis: gracePeriodCutoffMillis,
		timeFormatString:        timeFormatString,
		collectionMutex:         new(sync.Mutex),
		log:                     log,
	}, nil
}

func (s *streamingRITATimeIntervalWriter) newAutoFlushCollection(unixTSMillis int64,
	onFatal func(), autoFlushErrChan chan<- error) (*buffered.AutoFlushCollection, error) {

	//time.Unix(seconds, nanoseconds)
	//1000 milliseconds per second, 1000 nanosecodns to a microsecond. 1000 microseconds to a millisecond
	newTime := time.Unix(unixTSMillis/1000, (unixTSMillis%1000)*1000*1000).In(s.timezone)

	newColl, err := s.ritaDBManager.NewRITAOutputConnection(newTime.Format(s.timeFormatString))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start auto flusher for collection XXX-%s.%s", newTime.Format(s.timeFormatString), rita.RitaConnInputCollection)
	}
	err = s.ritaDBManager.EnsureMetaDBRecordExists(newColl.Database.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start auto flusher for collection XXX-%s.%s", newTime.Format(s.timeFormatString), rita.RitaConnInputCollection)
	}

	newAutoFlushCollection := buffered.NewAutoFlushCollection(newColl, s.collectionBufferSize, s.autoflushDeadline)
	started := newAutoFlushCollection.StartAutoFlush(autoFlushErrChan, onFatal)
	if !started {
		errmsg := fmt.Sprintf("failed to start auto flusher for collection XXX-%s.%s", newTime.Format(s.timeFormatString), rita.RitaConnInputCollection)
		return nil, errors.New(errmsg)
	}
	return newAutoFlushCollection, nil
}

func (s *streamingRITATimeIntervalWriter) initializeCurrentSegmentAndGracePeriod(
	onFatal func(), autoFlushErrChan chan<- error) error {
	currTime := s.clock.Now()
	currTimeMillis := currTime.UnixNano() / 1000000

	s.currentSegmentTS = s.segmentTSFactory.GetSegmentRelativeTimestamp(currTimeMillis)
	s.inGracePeriod = s.currentSegmentTS.OffsetFromSegmentStartMillis < s.gracePeriodCutoffMillis
	return nil
}

func (s *streamingRITATimeIntervalWriter) at(unixTSMillis int64) <-chan time.Time {
	currTime := s.clock.Now()
	currTimeMillis := currTime.UnixNano() / 1000000
	if currTimeMillis >= unixTSMillis {
		instantChan := make(chan time.Time, 1)
		instantChan <- currTime
		return instantChan
	}
	return s.clock.After(
		time.Duration(unixTSMillis-currTimeMillis) * time.Millisecond,
	)
}

func (s *streamingRITATimeIntervalWriter) getNextUpdateChan() <-chan time.Time {

	//Naively, we would need to lock over inGracePeriod and currentSegmentTS
	//However, this method is only called on the FlushLoop after modifications have
	//been made. We are only reading in this function and no one else is writing,
	//so we do not have to lock.

	if s.inGracePeriod {
		//update after the grace period expires
		return s.at(s.currentSegmentTS.SegmentStartMillis + s.gracePeriodCutoffMillis)
	}
	//update after the time segment changes
	return s.at(s.currentSegmentTS.SegmentStartMillis + s.currentSegmentTS.SegmentDurationMillis)
}

func (s *streamingRITATimeIntervalWriter) flushLoop(fatalContext context.Context,
	onFatal func(), wg *sync.WaitGroup, errsOut chan<- error) {

	updateChan := s.getNextUpdateChan()

FlushLoop:
	for {
		select {
		case <-fatalContext.Done():
			break FlushLoop
		case <-updateChan:

			//trash the time sent on the updateChan since the scheduler might
			//have been lazy and might have blocked us from getting to it instantly
			currTime := s.clock.Now()
			currTimeMillis := currTime.UnixNano() / 1000000

			s.collectionMutex.Lock()

			s.currentSegmentTS = s.segmentTSFactory.GetSegmentRelativeTimestamp(currTimeMillis)
			s.inGracePeriod = s.currentSegmentTS.OffsetFromSegmentStartMillis < s.gracePeriodCutoffMillis

			//assert that s.inGracePeriod toggled
			if s.inGracePeriod {
				//Beginning of grace period, different time segment

				//set previousCollection to currentCollection
				s.previousCollection = s.currentCollection

				//clear currentCollection so it is created when needed
				s.currentCollection = nil
				s.collectionMutex.Unlock()
			} else if s.previousCollection != nil {
				//End of grace period, same segment

				//unlock the mutex immediately since we don't want to
				//hold the lock while we flush a buffer.
				prevColl := s.previousCollection
				s.previousCollection = nil
				s.collectionMutex.Unlock()

				//Flush the previous collection and close it out
				prevColl.Flush()
				prevColl.Close()
				err := s.ritaDBManager.MarkImportFinishedInMetaDB(prevColl.Database())
				if err != nil {
					errsOut <- err
					break FlushLoop
				}
			} else {
				s.collectionMutex.Unlock()
			}

			//Note that getNextUpdateChan doesn't use the cached timestamp
			//in currentSegmentTS since it may take a decent amount of time
			//to flush a buffer to MongoDB
			updateChan = s.getNextUpdateChan()
		}
	}

	//This loop should only exit if there is an error (or the user shuts down the program)

	//wrap up the previous collection if it is open
	if s.previousCollection != nil {
		s.previousCollection.Flush()
		s.previousCollection.Close()
		/*
			BUG: https://github.com/activecm/ipfix-rita/issues/35
			err := s.ritaDBManager.MarkImportFinishedInMetaDB(s.previousCollection.Database())
			if err != nil {
				errsOut <- err
			}
		*/
	}

	//Wrap up the current collection
	if s.currentCollection != nil { //could be nil due to error
		s.currentCollection.Flush()
		s.currentCollection.Close()
		/*
			BUG: https://github.com/activecm/ipfix-rita/issues/35
			err := s.ritaDBManager.MarkImportFinishedInMetaDB(s.currentCollection.Database())
			if err != nil {
				errsOut <- err
			}
		*/
	}

	onFatal()
	wg.Done()
}

func (s *streamingRITATimeIntervalWriter) writeLoop(fatalContext context.Context,
	onFatal func(), wg *sync.WaitGroup, sessions <-chan *session.Aggregate, errsOut chan<- error) {

WriteLoop:
	for {
		select {
		case <-fatalContext.Done():
			break WriteLoop
		case sess, ok := <-sessions:
			if !ok { //how we know the program is shutting down
				break WriteLoop
			}

			sessEndMillis := sess.FlowEndMilliseconds()
			sessEndSegmentTS := s.segmentTSFactory.GetSegmentRelativeTimestamp(sessEndMillis)
			//ensure currentSegmentTS, inGracePeriod, currentCollection, and previousCollection
			//are consistent
			s.collectionMutex.Lock()

			//we drop the sameDuration check off the result from the next call
			//since we know we are only using a single segmentTSFactory
			segOffset, _ := s.currentSegmentTS.SegmentOffsetFrom(sessEndSegmentTS)

			if segOffset == 0 {
				var ritaConn parsetypes.Conn
				sess.ToRITAConn(&ritaConn, s.isIPLocal)

				if s.currentCollection == nil {
					var err error
					s.currentCollection, err = s.newAutoFlushCollection(s.currentSegmentTS.SegmentStartMillis, onFatal, errsOut)
					if err != nil {
						errsOut <- errors.Wrap(err, "could not lazily initialize MongoDB output collection")
						break WriteLoop
					}
				}

				//Insert into today's db
				err := s.currentCollection.Insert(ritaConn)
				if err != nil {
					errsOut <- errors.Wrap(err, "could not insert session into the current period collection")
					break WriteLoop
				}
			} else if segOffset == -1 && s.inGracePeriod {
				var ritaConn parsetypes.Conn
				sess.ToRITAConn(&ritaConn, s.isIPLocal)

				if s.previousCollection == nil {
					prevTimeMillis := s.currentSegmentTS.SegmentStartMillis - s.currentSegmentTS.SegmentDurationMillis

					var err error
					s.previousCollection, err = s.newAutoFlushCollection(prevTimeMillis, onFatal, errsOut)
					if err != nil {
						errsOut <- errors.Wrap(err, "could not lazily initialize MongoDB output collection")
						break WriteLoop
					}
				}

				//Insert into yesterday's db
				err := s.previousCollection.Insert(ritaConn)
				if err != nil {
					errsOut <- errors.Wrap(err, "could not insert session into the previous period collection")
					break WriteLoop
				}
			} else {
				s.log.Info("dropping out-of-time-segment session", logging.Fields{
					"session": fmt.Sprintf("%+v", sess),
				})
				//TODO: Add counters and track this
				//Drop the connection record
			}

			s.collectionMutex.Unlock()
		}
	}

	onFatal()
	wg.Done()
}

//Write starts the threads needed to back the streamingRITATimeIntervalWriter
//and sets up databases in MongoDB to hold the output sessions.
//Closing the sessions channel shuts down the threads. The error channel
//will close when the module has shut down.
func (s *streamingRITATimeIntervalWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	//initialize the current time derived variables (including MongoDB databases)
	errs := make(chan error)
	fatalContext, onFatal := context.WithCancel(context.Background())

	err := s.initializeCurrentSegmentAndGracePeriod(onFatal, errs)
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
		wg.Add(2)
		go s.flushLoop(fatalContext, onFatal, wg, errs)
		go s.writeLoop(fatalContext, onFatal, wg, sessions, errs)
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
