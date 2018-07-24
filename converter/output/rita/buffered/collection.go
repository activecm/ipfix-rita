package buffered

import (
	"sync"

	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
)

type Collection struct {
	mgoCollection *mgo.Collection
	buffer        []interface{}
	mutex         *sync.Mutex
}

func InitializeCollection(coll *Collection, mgoCollection *mgo.Collection, bufferSize int) {
	coll.mgoCollection = mgoCollection
	coll.buffer = make([]interface{}, 0, bufferSize)
	coll.mutex = new(sync.Mutex)
}

func (b *Collection) Insert(data interface{}) error {
	b.mutex.Lock()
	b.buffer = append(b.buffer, data)
	shouldFlush := len(b.buffer) == cap(b.buffer)
	b.mutex.Unlock()
	if shouldFlush {
		return b.Flush()
	}
	return nil
}

func (b *Collection) Flush() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.buffer) == 0 {
		return nil
	}
	bulk := b.mgoCollection.Bulk()
	bulk.Insert(b.buffer...)
	_, err := bulk.Run()
	if err != nil {
		return errors.Wrap(err, "could not perform bulk insert of output data into MongoDB")
	}
	b.buffer = b.buffer[:0]
	return nil
}

func (b *Collection) Close() error {
	err := b.Flush()
	b.mgoCollection.Database.Session.Close()
	return errors.Wrap(err, "could not close buffered.Collection")
}
