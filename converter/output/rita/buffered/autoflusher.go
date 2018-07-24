package buffered

import (
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"
)

type AutoFlushCollection struct {
	bufferedColl    Collection
	wg              *sync.WaitGroup
	errs            chan<- error
	stopChan        chan struct{}
	resetTimer      chan bool
	autoFlushTime   time.Duration
	autoFlushActive bool
}

func NewAutoFlushCollection(mgoCollection *mgo.Collection, bufferSize int,
	autoFlushTime time.Duration, errs chan<- error) *AutoFlushCollection {
	coll := &AutoFlushCollection{
		wg:              new(sync.WaitGroup),
		errs:            errs,
		stopChan:        make(chan struct{}),
		resetTimer:      make(chan bool, 1),
		autoFlushTime:   autoFlushTime,
		autoFlushActive: false,
	}
	InitializeCollection(&coll.bufferedColl, mgoCollection, bufferSize)
	return coll
}

func (b *AutoFlushCollection) StartAutoFlush() bool {
	if b.autoFlushActive {
		return false
	}
	b.wg.Add(1)
	go b.autoFlush()
	b.autoFlushActive = true
	return true
}

func (b *AutoFlushCollection) autoFlush() {
	timer := time.NewTimer(b.autoFlushTime)
Loop:
	for {
		select {
		case <-b.stopChan:
			break Loop
		case <-timer.C:

			//non blocking read to check the flag
			var shouldResetTimer bool
			select {
			case shouldResetTimer = <-b.resetTimer:
			default:
			}

			if !shouldResetTimer {
				err := b.bufferedColl.Flush()
				if err != nil {
					b.errs <- err
					continue //retry
				}
			}

			//we need to reset the timer whether or not we performed a flush
			//so we don't repeatedly flush
			timer.Reset(b.autoFlushTime)
		}
	}
	b.wg.Done()
}

func (b *AutoFlushCollection) Insert(data interface{}) {
	err := b.bufferedColl.Insert(data)
	if err != nil {
		b.errs <- err
		return
	}

	//non blocking send with a buffer to hold the flag
	select {
	case b.resetTimer <- true:
	default:
	}
}

func (b *AutoFlushCollection) Flush() {
	err := b.bufferedColl.Flush()
	if err != nil {
		b.errs <- err
		return
	}

	//non blocking send with a buffer to hold the flag
	select {
	case b.resetTimer <- true:
	default:
	}
}

func (b *AutoFlushCollection) Close() {
	//tell the autoflusher to stop
	close(b.stopChan)
	//wait for the autoflusher to stop
	b.wg.Wait()
	//close the underlying connection
	err := b.bufferedColl.Close()
	if err != nil {
		b.errs <- err
	}
}
