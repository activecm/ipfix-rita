package mongomatch

import (
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
	sessionsColl *mgo.Collection

	//flushing resources
	sessionsOut      chan<- *session.Aggregate
	preFlushMaxSize  int
	postFlushMaxSize int
	//diagnostic information
	nPacketConnsFlushed map[int]int
	oldConnsFlushed     int

	log logging.Logger
}

//NewMongoMatcher creates a new Matcher which is backed by a
//MongoDB collection
func NewMongoMatcher(db database.DB, log logging.Logger,
	sessionsOut chan<- *session.Aggregate,
	maxSize int, flushToPercent float32) (matching.Matcher, error) {

	sessionsCollection := db.NewCollection(SessionsCollName)

	err := sessionsCollection.EnsureIndex(mgo.Index{
		Key: []string{
			"IPAddressA", "transportPortA",
			"IPAddressB", "transportPortB",
			"protocolIdentifier", "exporter",
		},
		Name: "AggregateQuery",
	})

	if err != nil {
		sessionsCollection.Database.Session.Close()
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
		sessionsCollection.Database.Session.Close()
		return nil, errors.Wrap(err, "could not create ExpirationQuery index")
	}

	return &mongoMatcher{
		sessionsColl:        sessionsCollection,
		sessionsOut:         sessionsOut,
		preFlushMaxSize:     maxSize,
		postFlushMaxSize:    int(float32(maxSize)*flushToPercent + 0.5),
		nPacketConnsFlushed: make(map[int]int),
		oldConnsFlushed:     0,
		log:                 log,
	}, nil
}

func (m *mongoMatcher) Close() error {
	err := m.flushAll()
	if err != nil {
		return err
	}
	m.sessionsColl.Database.Session.Close()

	m.log.Info("mongo matcher exiting", logging.Fields{
		"1 packet flows left unstitched": m.nPacketConnsFlushed[1],
		"2 packet flows left unstitched": m.nPacketConnsFlushed[2],
		"other flows left unstitched":    m.oldConnsFlushed,
	})
	return nil
}

func (m *mongoMatcher) Find(aggQuery *session.AggregateQuery) session.Iterator {
	return newMongoSessionIterator(m.sessionsColl.Find(aggQuery).Iter())
}

func (m *mongoMatcher) Insert(sessAgg *session.Aggregate) error {
	//the MatcherID for mongoMatcher is just the MongoDB _id field
	//_id is set by the MongoDB server
	return m.sessionsColl.Insert(sessAgg)
}

func (m *mongoMatcher) Remove(sessAgg *session.Aggregate) error {
	return m.sessionsColl.RemoveId(sessAgg.MatcherID.(bson.ObjectId))
}

func (m *mongoMatcher) Update(sessAgg *session.Aggregate) error {
	return m.sessionsColl.UpdateId(sessAgg.MatcherID.(bson.ObjectId), sessAgg)
}

func (m *mongoMatcher) ShouldFlush() (bool, error) {
	count, err := m.sessionsColl.Count()
	if err != nil {
		return false, errors.Wrap(err, "could not check if the sessions collection is full")
	}
	return count >= m.preFlushMaxSize, nil
}

func (m *mongoMatcher) Flush() error {
	count, err := m.sessionsColl.Count()
	if err != nil {
		return errors.Wrap(err, "could not check if the sessions collection is empty")
	}
	if count <= m.preFlushMaxSize {
		return nil
	}

	for i := 1; i <= 2; i++ {
		//flush out the garbage first
		err = m.flushNPacketConnections(i, &count)
		if err != nil {
			return errors.Wrapf(err,
				"failed to flush %d packet connections from the sessions collection\n"+
					"flush started at: %d\n"+
					"current count: %d\n"+
					"target count: %d", i, m.preFlushMaxSize, count, m.postFlushMaxSize,
			)
		}

		//If we've flushed enough flows, return
		if count <= m.postFlushMaxSize {
			return nil
		}
	}
	//flush enough old flows to get to the postFlushMaxSize
	err = m.flushOldest(&count, m.postFlushMaxSize)

	return errors.Wrapf(err,
		"failed to flush oldest connections from the sessions collection\n"+
			"flush started at: %d\n"+
			"current count: %d\n"+
			"target count: %d", m.preFlushMaxSize, count, m.postFlushMaxSize,
	)
}

//flushAll flushes the entirety of the sessionsColl
func (m *mongoMatcher) flushAll() error {
	count, err := m.sessionsColl.Count()
	if err != nil {
		return errors.Wrap(err, "could not check if the sessions collection is empty")
	}
	if count == 0 {
		return nil
	}

	for i := 1; i <= 2; i++ {
		err = m.flushNPacketConnections(i, &count)
		if err != nil {
			return errors.Wrapf(err,
				"failed to flush %d packet connections from the sessions collection\n"+
					"flush started at: %d\n"+
					"current count: %d\n"+
					"target count: %d", i, m.preFlushMaxSize, count, m.postFlushMaxSize,
			)
		}
	}
	err = m.flushOldest(&count, 0)

	return errors.Wrapf(err,
		"failed to flush oldest connections from the sessions collection\n"+
			"flush started at: %d\n"+
			"current count: %d\n"+
			"target count: %d", m.preFlushMaxSize, count, m.postFlushMaxSize,
	)
}

//flushNPacketConnections flushes sessions which contain
//exactly n packets, decrementing currentCount as the session
//aggregates are removed
func (m *mongoMatcher) flushNPacketConnections(n int, currentCount *int) error {

	flushIter := m.sessionsColl.Find(bson.M{
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
		err := m.sessionsColl.RemoveId(sessAgg.MatcherID.(bson.ObjectId))
		if err != nil {
			return errors.Wrapf(err, "could not remove session from sessions collection\n%+v", sessAgg)
		}
		*currentCount--
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
func (m *mongoMatcher) flushOldest(currentCount *int, targetCount int) error {
	flushIter := m.sessionsColl.Find(nil).Sort("_id").Iter()

	sessAgg := new(session.Aggregate)
	for flushIter.Next(sessAgg) && *currentCount > targetCount {
		err := m.sessionsColl.RemoveId(sessAgg.MatcherID.(bson.ObjectId))
		if err != nil {
			return errors.Wrapf(err, "could not remove session from sessions collection\n%+v", sessAgg)
		}
		*currentCount--
		m.oldConnsFlushed++

		m.sessionsOut <- sessAgg
		sessAgg = new(session.Aggregate)
	}
	return errors.Wrapf(flushIter.Err(), "could not find %d old sessions to flush", *currentCount-targetCount)
}
