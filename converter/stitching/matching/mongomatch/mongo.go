package mongomatch

import (
	"sync/atomic"

	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/stitching/matching"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//SessionsCollName is the name of the collection used to back
//a mongoMatcher
const SessionsCollName = "sessions"

//mongoSessionIterator wraps *mgo.Iter to provide the
//appropriate signature to implement session.Iterator
type mongoSessionIterator struct {
	mgoIter *mgo.Iter
}

func newMongoSessionIterator(mgoIter *mgo.Iter) session.Iterator {
	return mongoSessionIterator{
		mgoIter: mgoIter,
	}
}

func (m mongoSessionIterator) Next(sessAgg *session.Aggregate) bool {
	return m.mgoIter.Next(sessAgg) //adapt interface{}
}

func (m mongoSessionIterator) Err() error {
	return m.mgoIter.Err()
}

//mongoMatcher uses a MongoDB collection to implement
//matching.Matcher
type mongoMatcher struct {
	sessionsCollConnections []*mgo.Collection
	currSession             int
	currCount               int64
	//flushing resources
	sessionsOut      chan<- *session.Aggregate
	preFlushMaxSize  int64
	postFlushMaxSize int64
	//diagnostic information
	nPacketConnsFlushed map[int]int
	oldConnsFlushed     int

	log logging.Logger
}

//NewMongoMatcher creates a new Matcher which is backed by a
//MongoDB collection
func NewMongoMatcher(db database.DB, log logging.Logger,
	sessionsOut chan<- *session.Aggregate, numStitchers int32,
	maxSize int64, flushToPercent float32) (matching.Matcher, error) {

	//create a pool of database connections
	var sessionsCollConnections []*mgo.Collection
	for i := 0; i < int(numStitchers); i++ {
		sessionsCollConnections = append(
			sessionsCollConnections,
			db.NewCollection(SessionsCollName),
		)
	}

	//grab a connection to initialize the database
	sessionsCollection := sessionsCollConnections[0]

	err := sessionsCollection.EnsureIndex(mgo.Index{
		Key: []string{
			"IPAddressA", "transportPortA",
			"IPAddressB", "transportPortB",
			"protocolIdentifier", "exporter",
		},
		Name: "AggregateQuery",
	})

	if err != nil {
		for i := range sessionsCollConnections {
			sessionsCollConnections[i].Database.Session.Close()
		}
		return nil, errors.Wrap(err, "could not create AggregateQuery index")
	}

	err = sessionsCollection.EnsureIndex(mgo.Index{
		Key: []string{
			"packetTotalCountAB",
			"packetTotalCountBA",
		},
		Name: "ExpirationQuery",
	})

	if err != nil {
		for i := range sessionsCollConnections {
			sessionsCollConnections[i].Database.Session.Close()
		}
		return nil, errors.Wrap(err, "could not create ExpirationQuery index")
	}

	currCount, err := sessionsCollection.Count()
	if err != nil {
		for i := range sessionsCollConnections {
			sessionsCollConnections[i].Database.Session.Close()
		}
		return nil, errors.Wrap(err, "could not count records in sessions collection")
	}

	return &mongoMatcher{
		sessionsCollConnections: sessionsCollConnections,
		currSession:             0,
		currCount:               int64(currCount),
		sessionsOut:             sessionsOut,
		preFlushMaxSize:         maxSize,
		postFlushMaxSize:        int64(float32(maxSize)*flushToPercent + 0.5),
		nPacketConnsFlushed:     make(map[int]int),
		oldConnsFlushed:         0,
		log:                     log,
	}, nil
}

func (m *mongoMatcher) Close() error {
	err := m.flushTo(0)
	if err != nil {
		return err
	}
	for i := range m.sessionsCollConnections {
		m.sessionsCollConnections[i].Database.Session.Close()
	}

	m.log.Info("mongo matcher exiting", logging.Fields{
		"1 packet flows left unstitched": m.nPacketConnsFlushed[1],
		"2 packet flows left unstitched": m.nPacketConnsFlushed[2],
		"other flows left unstitched":    m.oldConnsFlushed,
	})
	return nil
}

func (m *mongoMatcher) Find(aggQuery *session.AggregateQuery) session.Iterator {
	return newMongoSessionIterator(m.getNextSessionsCollConnection().Find(aggQuery).Iter())
}

func (m *mongoMatcher) Insert(sessAgg *session.Aggregate) error {
	//the MatcherID for mongoMatcher is just the MongoDB _id field
	//_id is set by the MongoDB server
	err := m.getNextSessionsCollConnection().Insert(sessAgg)
	if err != nil {
		return errors.Wrapf(err, "could not insert %+v", sessAgg)
	}
	atomic.AddInt64(&m.currCount, 1)
	return nil
}

func (m *mongoMatcher) Remove(sessAgg *session.Aggregate) error {
	err := m.getNextSessionsCollConnection().RemoveId(sessAgg.MatcherID.(bson.ObjectId))
	if err != nil {
		return errors.Wrapf(err, "could not remove %+v", sessAgg)
	}
	atomic.AddInt64(&m.currCount, -1)
	return nil
}

func (m *mongoMatcher) Update(sessAgg *session.Aggregate) error {
	err := m.getNextSessionsCollConnection().UpdateId(sessAgg.MatcherID.(bson.ObjectId), sessAgg)
	return errors.Wrapf(err, "could not update %+v", sessAgg)
}

func (m *mongoMatcher) ShouldFlush() (bool, error) {
	return atomic.LoadInt64(&m.currCount) >= m.preFlushMaxSize, nil
}

func (m *mongoMatcher) Flush() error {
	return m.flushTo(m.postFlushMaxSize)
}

func (m *mongoMatcher) flushTo(targetCount int64) error {
	startCount := atomic.LoadInt64(&m.currCount)
	if startCount <= targetCount {
		return nil
	}
	defer func() {
		m.log.Info("finished session aggregate flush", logging.Fields{
			"start count":   startCount,
			"current count": atomic.LoadInt64(&m.currCount),
			"target count":  targetCount,
		})
	}()

	for i := 1; i <= 2; i++ {
		//flush out the garbage first
		err := m.flushNPacketConnections(i)
		if err != nil {
			return errors.Wrapf(err,
				"failed to flush %d packet connections from the sessions collection\n"+
					"flush started at: %d\n"+
					"current count: %d\n"+
					"target count: %d", i, startCount, atomic.LoadInt64(&m.currCount), m.postFlushMaxSize,
			)
		}

		//If we've flushed enough flows, return
		if atomic.LoadInt64(&m.currCount) <= targetCount {
			return nil
		}
	}
	//flush enough old flows to get to the postFlushMaxSize
	err := m.flushOldest(targetCount)

	return errors.Wrapf(err,
		"failed to flush oldest connections from the sessions collection\n"+
			"flush started at: %d\n"+
			"current count: %d\n"+
			"target count: %d", startCount, atomic.LoadInt64(&m.currCount), targetCount,
	)
}

//flushNPacketConnections flushes sessions which contain
//exactly n packets, decrementing currentCount as the session
//aggregates are removed
func (m *mongoMatcher) flushNPacketConnections(n int) error {

	flushIter := m.getNextSessionsCollConnection().Find(bson.M{
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
		err := m.Remove(sessAgg)
		if err != nil {
			return errors.Wrap(err, "could not flush session aggregate")
		}
		m.nPacketConnsFlushed[n]++

		m.sessionsOut <- sessAgg
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
func (m *mongoMatcher) flushOldest(targetCount int64) error {
	flushIter := m.getNextSessionsCollConnection().Find(nil).Sort("_id").Iter()

	sessAgg := new(session.Aggregate)
	for flushIter.Next(sessAgg) && atomic.LoadInt64(&m.currCount) > targetCount {
		m.Remove(sessAgg)
		m.oldConnsFlushed++

		m.sessionsOut <- sessAgg
		sessAgg = new(session.Aggregate)
	}
	return errors.Wrapf(flushIter.Err(), "could not find %d old sessions to flush", atomic.LoadInt64(&m.currCount)-targetCount)
}

func (m *mongoMatcher) getNextSessionsCollConnection() *mgo.Collection {
	conn := m.sessionsCollConnections[m.currSession]
	m.currSession = (m.currSession + 1) % len(m.sessionsCollConnections)
	return conn
}
