package buffered

import (
	"sync"
	"time"

	mgo "github.com/globalsign/mgo"
)

//AutoFlushCollection wraps a Collection and ensures
//the data in the Collection's buffer is flushed to MongoDB
//within a deadline.
type AutoFlushCollection struct {
	bufferedColl         Collection
	wg                   *sync.WaitGroup
	autoFlushErrChan     chan<- error
	stopChan             chan struct{}
	resetTimer           chan bool
	deadlineInterval     time.Duration
	autoFlushActive      bool
	autoFlushActiveMutex *sync.Mutex
}

//NewAutoFlushCollection creates a AutoFlushCollection which
//wraps a *mgo.Collection with with a buffer for insertions and
//ensures the buffer is written to MongoDB within a deadline.
//The deadline is pushed back to time.Now() + deadlineInterval
//everytime Insert or Flush is called.
func NewAutoFlushCollection(mgoCollection *mgo.Collection, bufferSize int64,
	deadlineInterval time.Duration, autoFlushErrChan chan<- error) *AutoFlushCollection {
	coll := &AutoFlushCollection{
		wg:                   new(sync.WaitGroup),
		autoFlushErrChan:     autoFlushErrChan,
		stopChan:             make(chan struct{}),
		resetTimer:           make(chan bool, 1),
		deadlineInterval:     deadlineInterval,
		autoFlushActive:      false,
		autoFlushActiveMutex: new(sync.Mutex),
	}
	InitializeCollection(&coll.bufferedColl, mgoCollection, bufferSize)
	return coll
}

//Database returns the name of the database the collection is in
func (b *AutoFlushCollection) Database() string {
	return b.bufferedColl.Database()
}

//Name returns the name of the underlying MongoDB collection
func (b *AutoFlushCollection) Name() string {
	return b.bufferedColl.Name()
}

//StartAutoFlush starts the go routine which ensures the
//AutoFlushCollection's buffer is flushed out within a deadline.
//autoFlushStopped will be called once the auto flusher has exited
func (b *AutoFlushCollection) StartAutoFlush(autoFlushStopped func()) bool {
	b.autoFlushActiveMutex.Lock()
	defer b.autoFlushActiveMutex.Unlock()
	if b.autoFlushActive {
		return false
	}
	b.wg.Add(1)
	go b.autoFlush(autoFlushStopped)
	b.autoFlushActive = true
	return true
}

func (b *AutoFlushCollection) autoFlush(autoFlushStopped func()) {
	timer := time.NewTimer(b.deadlineInterval)
Loop:
	for {
		select {
		case <-b.stopChan:
			break Loop
		case <-b.resetTimer:
			timer.Reset(b.deadlineInterval)
		case <-timer.C:
			err := b.Flush()
			if err != nil {
				b.autoFlushErrChan <- err
				break Loop
			}
		}
	}
	b.wg.Done()
	b.autoFlushActiveMutex.Lock()
	b.autoFlushActive = false
	b.autoFlushActiveMutex.Unlock()
	autoFlushStopped()
}

//Insert writes a record into the Collection's buffer.
//If the buffer is full after the insertion, Flush is called.
func (b *AutoFlushCollection) Insert(data interface{}) error {
	err := b.bufferedColl.Insert(data)
	if err != nil {
		return err
	}

	//non blocking send with a buffer to hold the flag
	select {
	case b.resetTimer <- true:
	default:
	}
	return nil
}

//Flush sends the data inside the Collection's buffer to MongoDB
func (b *AutoFlushCollection) Flush() error {
	err := b.bufferedColl.Flush()
	if err != nil {
		return err
	}

	//non blocking send with a buffer to hold the flag
	select {
	case b.resetTimer <- true:
	default:
	}
	return nil
}

//Close closes the socket wrapped by the Collection
func (b *AutoFlushCollection) Close() error {
	//tell the autoflusher to stop
	close(b.stopChan)
	//wait for the autoflusher to stop
	b.wg.Wait()
	//close the underlying connection
	err := b.bufferedColl.Close()
	return err
}
