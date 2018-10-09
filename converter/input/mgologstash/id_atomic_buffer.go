package mgologstash

import (
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/pkg/errors"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

//idAtomicBuffer is not used within ipfix-rita. However, it does
//show how to implement a mgologstash.Buffer. It is used in
//reader_test as it is the simplest buffer implementation.

//idAtomicBuffer works by selecting and removing the least recently inserted
//record in an input collection
type idAtomicBuffer struct {
	input *mgo.Collection
	err   error
	log   logging.Logger
}

//NewIDAtomicBuffer returns an mgologstash.Buffer which pulls
//input records atomically from MongoDB in (roughly) insertion order
func NewIDAtomicBuffer(input *mgo.Collection, log logging.Logger) Buffer {
	return &idAtomicBuffer{
		input: input,
		log:   log,
	}
}

//Next returns the next record that was inserted into the input collection.
//Next returns false if there is no more data. Next may set an error when
//it returns false. This error can be read with Err()
func (b *idAtomicBuffer) Next(out *Flow) bool {

	getNextRecord := true
	for getNextRecord {
		var input bson.M
		_, err := b.input.Find(nil).Sort("_id").Apply(
			mgo.Change{
				Remove: true,
			},
			&input,
		)

		if err != nil {
			if err != mgo.ErrNotFound {
				b.err = errors.Wrap(err, "could not fetch next record from input collection")
			}
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
func (b *idAtomicBuffer) Err() error {
	return b.err
}

//Close closes the socket to the MongoDB server
func (b *idAtomicBuffer) Close() {
	b.input.Database.Session.Close()
}
