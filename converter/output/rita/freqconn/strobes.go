package freqconn

import (
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/activecm/ipfix-rita/converter/output/rita/constants"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

//StrobesNotifier implements ConnCountNotifier and serves to keep
//the RITA conn and freqConn collections in line with the internal
//connection count map. This effectively implements RITA's "strobes" analysis.
type StrobesNotifier struct {
	db                *mgo.Database
	connAutoFlushColl *buffered.AutoFlushCollection
}

//NewStrobesNotifier creates a new StrobesNotifier from a MongoDB
//database handle. If an AutoFlushCollection is currently bound
//to the RITA conn collection, pass in a reference and the Notifier
//will ensure the collection buffer is flushed before altering the
//conn collection. If no such AutoFlushCollection exists, pass in nil.
// Note the StrobesNotifier.Close() method
//closes the socket used by the db handle. You may want to
//copy the initial connection before passing the handle to this
//constructor.
func NewStrobesNotifier(db *mgo.Database, connAutoFlushColl *buffered.AutoFlushCollection) StrobesNotifier {
	return StrobesNotifier{
		db:                db,
		connAutoFlushColl: connAutoFlushColl,
	}
}

//LoadFreqConnCollection reads the data in the StrobesCollection
//of a RITA database into a map which counts how many times
//a connection pair was seen
func (s StrobesNotifier) LoadFreqConnCollection() (map[UConnPair]int, error) {
	strobeIter := s.db.C(constants.StrobesCollection).Find(nil).Iter()
	dataMap := make(map[UConnPair]int)
	var entry FreqConn
	for strobeIter.Next(&entry) {
		dataMap[entry.UConnPair] = entry.ConnectionCount
	}
	err := strobeIter.Err()
	return dataMap, err
}

//ThresholdMet deletes any matching entries in the RITA ConnCollection
//and creates a new record in the freqConn collection
func (s StrobesNotifier) ThresholdMet(connPair UConnPair, count int) error {

	//We have to ensure that the auto flush collection pushes any buffered
	//data to the conn collection before we issue the remove command.
	//Otherwise data may be pushed into the conn collection after we call
	//remove.
	if s.connAutoFlushColl != nil {
		err := s.connAutoFlushColl.Flush()
		if err != nil {
			return err
		}
	}

	_, err := s.db.C(constants.ConnCollection).RemoveAll(bson.M{
		"$and": []bson.M{
			bson.M{"id_orig_h": connPair.Src},
			bson.M{"id_resp_h": connPair.Dst},
		},
	})

	if err != nil {
		return err
	}
	err = s.db.C(constants.StrobesCollection).Insert(FreqConn{
		UConnPair:       connPair,
		ConnectionCount: count,
	})
	return err
}

//ThresholdExceeded updates the connection_count field in a freqConn collection
//entry matching a given UConnPair
func (s StrobesNotifier) ThresholdExceeded(connPair UConnPair, count int) error {
	//Note we have to track the count in counter.go anyways, so
	//we could just update with count instead of calling inc
	//but inc gets the point across a bit better.

	err := s.db.C(constants.StrobesCollection).Update(
		bson.M{
			"src": connPair.Src,
			"dst": connPair.Dst,
		},
		bson.M{
			"$inc": bson.M{
				"connection_count": 1,
			},
		},
	)
	return err
}

//Close closes the socket wrapped by the Database
func (s StrobesNotifier) Close() {
	s.db.Session.Close()
}
