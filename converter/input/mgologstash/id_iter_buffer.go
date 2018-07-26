package mgologstash

import (
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//idIterBuffer is not used within ipfix-rita. However, it does
//show how to implement a mgologstash.Buffer

//idIterBuffer works by selecting and removing the least recently inserted
//record in an input collection
type idIterBuffer struct {
	input *mgo.Collection
	iter  *mgo.Iter
	err   error
	log   logging.Logger
}

//NewIDIterBuffer returns an mgologstash.Buffer which pulls
//input records one by one from MongoDB in (roughly) insertion order
func NewIDIterBuffer(input *mgo.Collection, log logging.Logger) Buffer {
	return &idIterBuffer{
		input: input,
		log:   log,
	}
}

//Next returns the next record that was inserted into the input collection.
//Next returns false if there is no more data. Next may set an error when
//it returns false. This error can be read with Err()
func (b *idIterBuffer) Next(out *Flow) bool {
	if b.iter == nil || b.iter.Done() {
		b.iter = b.input.Find(nil).Sort("_id").Iter()
	}

	getNextRecord := true
	for getNextRecord {

		var input bson.M
		ok := b.iter.Next(&input)
		if b.iter.Err() != nil {
			if b.iter.Err() != mgo.ErrNotFound {
				b.err = errors.Wrap(b.iter.Err(), "could not fetch next record from input collection")
			}
			return false
		}
		if !ok {
			return false
		}

		err := b.input.RemoveId(input["_id"].(bson.ObjectId))
		if err != nil {
			b.err = errors.Wrap(err, "could not remove record from input collection")
			return false
		}

		err = out.FillFromBSONMap(input)
		if err == nil {
			getNextRecord = false
		} else {
			b.log.Error(err, logging.Fields{"inputMap": input})
		}
	}

	return true
}

//Err returns any errors set by Read()
func (b *idIterBuffer) Err() error {
	return b.err
}

//Close closes the socket to the MongoDB server
func (b *idIterBuffer) Close() {
	b.input.Database.Session.Close()
}
