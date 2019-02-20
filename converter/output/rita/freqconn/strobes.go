package freqconn

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

//StrobesCollection contains the name for the RITA freqConn MongoDB collection
const StrobesCollection = "freqConn"

//ConnCollection contains the name for the RITA conn MongoDB collection
const ConnCollection = "conn"

//LoadFreqConnCollection reads the data in the StrobesCollection
//of a RITA database into a map which counts how many times
//a connection pair was seen
func LoadFreqConnCollection(db *mgo.Database) (map[UConnPair]int, error) {
	strobeIter := db.C(StrobesCollection).Find(nil).Iter()
	dataMap := make(map[UConnPair]int)
	var entry FreqConn
	for strobeIter.Next(&entry) {
		dataMap[entry.UConnPair] = entry.ConnectionCount
	}
	err := strobeIter.Err()
	return dataMap, err
}

//GetFreqInserter returns a method for use with ConnCounter's
//thresholdMetFunc which deletes any matching entries in the RITA ConnCollection
//and creates a new record in the freqConn collection
func GetFreqInserter(db *mgo.Database) func(UConnPair, int) error {
	return func(connPair UConnPair, count int) error {
		_, err := db.C(ConnCollection).RemoveAll(bson.M{
			"$and": []bson.M{
				bson.M{"id_orig_h": connPair.Src},
				bson.M{"id_resp_h": connPair.Dst},
			},
		})

		if err != nil {
			return err
		}
		err = db.C(StrobesCollection).Insert(FreqConn{
			UConnPair:       connPair,
			ConnectionCount: count,
		})
		return err
	}
}

//GetFreqIncrementer returns a method for use with ConnCounter's
//thresholdExceededFunc which sets the connection_count matching
//a given UConnPair
func GetFreqIncrementer(db *mgo.Database) func(UConnPair, int) error {
	return func(connPair UConnPair, count int) error {

		//Note we have to track the count in counter.go anyways, so
		//we could just update with count instead of calling inc
		//but inc gets the point across a bit better.

		err := db.C(StrobesCollection).Update(
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
}
