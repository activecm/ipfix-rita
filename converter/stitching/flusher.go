package stitching

import (
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//flusher ensures the sessionsColl never exceeds
//a given number of session aggregates (preFlushMaxSize)
//
//Any records flushed by the flusher are sent to the output
//channel as is.
//
//The flusher is responsible for flushing out session aggregates
//which were never matched in the opposite direction.
type flusher struct {
	sessionsColl     *mgo.Collection
	sessionsOut      chan<- *session.Aggregate
	preFlushMaxSize  int
	postFlushMaxSize int
	//diagnostic information
	nPacketConnsFlushed map[int]int
	oldConnsFlushed     int
}

//newFlusher creates a new flusher. The flusher will begin a flush
//when the sessionsColl reaches maxSize and flush enough records
//to ensure the sessionsColl contains at most maxSize * flushToPercent records
func newFlusher(sessionsColl *mgo.Collection, sessionsOut chan<- *session.Aggregate,
	maxSize int, flushToPercent float32) *flusher {
	return &flusher{
		sessionsColl:        sessionsColl,
		sessionsOut:         sessionsOut,
		preFlushMaxSize:     maxSize,
		postFlushMaxSize:    int(float32(maxSize)*flushToPercent + 0.5),
		nPacketConnsFlushed: make(map[int]int),
		oldConnsFlushed:     0,
	}
}

//close closes the underlying connection to MongoDB
func (f *flusher) close() {
	f.sessionsColl.Database.Session.Close()
}

//shouldFlush checks if the sessionsColl contains
//postFlushMaxSize records or more
func (f *flusher) shouldFlush() (bool, error) {
	count, err := f.sessionsColl.Count()
	if err != nil {
		return false, errors.Wrap(err, "could not check if the sessions collection is full")
	}
	return count >= f.preFlushMaxSize, nil
}

//flush removes enough records from the sessionsColl
//to ensure there are at most postFlushMaxSize records in the collection
func (f *flusher) flush() error {
	count, err := f.sessionsColl.Count()
	if err != nil {
		return errors.Wrap(err, "could not check if the sessions collection is empty")
	}
	if count <= f.preFlushMaxSize {
		return nil
	}

	for i := 1; i <= 2; i++ {
		//flush out the garbage first
		err := f.flushNPacketConnections(i, &count)
		if err != nil {
			return errors.Wrapf(err,
				"failed to flush %d packet connections from the sessions collection\n"+
					"flush started at: %d\n"+
					"current count: %d\n"+
					"target count: %d", i, f.preFlushMaxSize, count, f.postFlushMaxSize,
			)
		}

		//If we've flushed enough flows, return
		if count <= f.postFlushMaxSize {
			return nil
		}
	}
	//flush enough old flows to get to the postFlushMaxSize
	err = f.flushOldest(&count, f.postFlushMaxSize)

	return errors.Wrapf(err,
		"failed to flush oldest connections from the sessions collection\n"+
			"flush started at: %d\n"+
			"current count: %d\n"+
			"target count: %d", f.preFlushMaxSize, count, f.postFlushMaxSize,
	)
}

//flushAll flushes the entirety of the sessionsColl
func (f *flusher) flushAll() error {
	count, err := f.sessionsColl.Count()
	if err != nil {
		return errors.Wrap(err, "could not check if the sessions collection is empty")
	}
	if count == 0 {
		return nil
	}

	for i := 1; i <= 2; i++ {
		err := f.flushNPacketConnections(i, &count)
		if err != nil {
			return errors.Wrapf(err,
				"failed to flush %d packet connections from the sessions collection\n"+
					"flush started at: %d\n"+
					"current count: %d\n"+
					"target count: %d", i, f.preFlushMaxSize, count, f.postFlushMaxSize,
			)
		}
	}
	err = f.flushOldest(&count, 0)
	return errors.Wrapf(err,
		"failed to flush oldest connections from the sessions collection\n"+
			"flush started at: %d\n"+
			"current count: %d\n"+
			"target count: %d", f.preFlushMaxSize, count, f.postFlushMaxSize,
	)
}

//flushNPacketConnections flushes sessions which contain
//exactly n packets, decrementing currentCount as the session
//aggregates are removed
func (f *flusher) flushNPacketConnections(n int, currentCount *int) error {

	flushIter := f.sessionsColl.Find(bson.M{
		"$or": []bson.M{
			bson.M{
				"packetTotalCountAB": n,
				"packetTotalCountBA": 0,
			},
			bson.M{
				"packetTotalCountAB": n,
				"packetTotalCountBA": 0,
			},
		},
	}).Iter()

	sessAgg := new(session.Aggregate)
	for flushIter.Next(sessAgg) {
		err := f.sessionsColl.RemoveId(sessAgg.ID)
		if err != nil {
			return errors.Wrapf(err, "could not remove session from sessions collection\n%+v", sessAgg)
		}
		*currentCount--
		f.nPacketConnsFlushed[n]++

		f.sessionsOut <- sessAgg
		sessAgg = new(session.Aggregate)
	}
	return errors.Wrapf(flushIter.Err(), "could not find all %d packet sessions to flush", n)
}

//flushOldest flushes records out of the sessionsColl
//until the sessionsColl contains targetCount session aggregates.
//
//flushOldest prioritizes records by how long they have sat in the
//collection as determined by the timestamp that is part of
//MongoDB's ObjectId
func (f *flusher) flushOldest(currentCount *int, targetCount int) error {
	flushIter := f.sessionsColl.Find(nil).Sort("_id").Iter()

	sessAgg := new(session.Aggregate)
	for flushIter.Next(sessAgg) && *currentCount > targetCount {
		err := f.sessionsColl.RemoveId(sessAgg.ID)
		if err != nil {
			return errors.Wrapf(err, "could not remove session from sessions collection\n%+v", sessAgg)
		}
		*currentCount--
		f.oldConnsFlushed++

		f.sessionsOut <- sessAgg
		sessAgg = new(session.Aggregate)
	}
	return errors.Wrapf(flushIter.Err(), "could not find %d old sessions to flush", *currentCount-targetCount)
}
