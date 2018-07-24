package mgologstash

import (
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//idBulkBuffer works by selecting and removing the least recently inserted
//record in an input collection
type idBulkBuffer struct {
	input     *mgo.Collection
	buffer    []bson.M
	readIndex int
	err       error
	log       logging.Logger
}

//NewIDBulkBuffer returns an ipfix.Buffer backed by MongoDB and fed by Logstash
func NewIDBulkBuffer(input *mgo.Collection, bufferSize int, log logging.Logger) Buffer {
	return &idBulkBuffer{
		input:  input,
		buffer: make([]bson.M, 0, bufferSize),
		log:    log,
	}
}

//Next returns the next record that was inserted into the input collection.
//Next returns false if there is no more data. Next may set an error when
//it returns false. This error can be read with Err()
func (b *idBulkBuffer) Next(out *Flow) bool {
	getNextRecord := true
	for getNextRecord {
		if b.readIndex == len(b.buffer) {
			//clear the buffer
			b.buffer = b.buffer[:0]

			//refill the buffer
			err := b.input.Find(nil).Sort("_id").Batch(len(b.buffer)).Limit(len(b.buffer)).All(&b.buffer)
			if err != nil {
				if err != mgo.ErrNotFound {
					b.err = errors.Wrap(err, "could not fetch next batch of records from input collection")
				}
				return false
			}

			//remove the elements that have been transferred to the buffer
			bulkRemove := b.input.Bulk()
			for i := range b.buffer {
				bulkRemove.Remove(bson.M{"_id": b.buffer[i]["_id"].(bson.ObjectId)})
			}
			_, err = bulkRemove.Run()
			if err != nil {
				b.err = errors.Wrap(err, "could not remove next batch of records from input collection")
				return false
			}

			//reset the read index
			b.readIndex = 0

			//nothing found
			if len(b.buffer) == 0 {
				return false
			}
		}

		inputMap := b.buffer[b.readIndex]
		b.readIndex++
		err := out.FillFromBSONMap(inputMap)
		if err == nil {
			getNextRecord = false
		} else {
			b.log.Error(err, logging.Fields{"inputMap": inputMap})
		}
	}

	return true
}

//Err returns any errors set by Read()
func (b *idBulkBuffer) Err() error {
	return b.err
}

//Close closes the socket to the MongoDB server
func (b *idBulkBuffer) Close() {
	b.input.Database.Session.Close()
}
