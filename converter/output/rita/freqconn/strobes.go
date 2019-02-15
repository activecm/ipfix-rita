package freqconn

import (
	"github.com/globalsign/mgo"
	"gopkg.in/mgo.v2/bson"
)

const strobesCollection = "freqConn"
const connCollection = "conn"

//LoadFreqConnCollection reads the data in the strobesCollection
//of a RITA database into a map which counts how many times
//a connection pair was seen
func LoadFreqConnCollection(db *mgo.Database) (map[UConnPair]int, error) {
	strobeIter := db.C(strobesCollection).Find(nil).Iter()
	dataMap := make(map[UConnPair]int)
	var entry FreqConn
	for strobeIter.Next(&entry) {
		dataMap[entry.UConnPair] = entry.ConnectionCount
	}
	err := strobeIter.Err()
	return dataMap, err
}

//GetFreqInserter returns a method for use with ConnCounter's
//thresholdMetFunc which deletes any matching entries in the RITA conn collection
//and creates a new record in the freqConn collection
func GetFreqInserter(db *mgo.Database) func(UConnPair, int) error {
	return func(connPair UConnPair, count int) error {
		err := db.C(connCollection).Remove(bson.M{
			"$and": []bson.M{
				bson.M{"id_orig_h": connPair.Src},
				bson.M{"id_resp_h": connPair.Dst},
			},
		})
		if err != nil {
			return err
		}
		err = db.C(strobesCollection).Insert(FreqConn{
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
		err := db.C(strobesCollection).Update(
			bson.M{
				"src": connPair.Src,
				"dst": connPair.Dst,
			},
			bson.M{
				"connection_count": count,
			},
		)
		return err
	}
}
