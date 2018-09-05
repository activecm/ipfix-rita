package buffered

import (
	"sync"

	"github.com/pkg/errors"
	mgo "github.com/globalsign/mgo"
)

//Collection wraps an *mgo.Collection in order
//to provide buffered insertion.
type Collection struct {
	mgoCollection *mgo.Collection
	buffer        []interface{}
	mutex         *sync.Mutex
}

//InitializeCollection wraps a *mgo.Collection with a buffer of a given size
//for performing buffered insertions.
func InitializeCollection(coll *Collection, mgoCollection *mgo.Collection, bufferSize int64) {
	coll.mgoCollection = mgoCollection
	coll.buffer = make([]interface{}, 0, bufferSize)
	coll.mutex = new(sync.Mutex)
}

//Database returns the name of the database the collection is in
func (b *Collection) Database() string {
	return b.mgoCollection.Database.Name
}

//Name returns the name of the underlying MongoDB collection
func (b *Collection) Name() string {
	return b.mgoCollection.Name
}

//Insert writes a record into the Collection's buffer.
//If the buffer is full after the insertion, Flush is called.
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

//Flush sends the data inside the Collection's buffer to MongoDB
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

//Close closes the socket wrapped by the Collection
func (b *Collection) Close() error {
	err := b.Flush()
	b.mgoCollection.Database.Session.Close()
	return errors.Wrap(err, "could not close buffered.Collection")
}
