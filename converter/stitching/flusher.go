package stitching

import (
	"container/list"
	"context"
	"math"
	"sync"

	"github.com/activecm/ipfix-rita/converter/stitching/session"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type expireTimeList struct {
	mutex       *sync.Mutex
	expireTimes *list.List
}

type flusher struct {
	exporterAddress       string
	maxExpireTimeMap      map[int]expireTimeList
	maxExpireTimeMapMutex *sync.RWMutex
	minChangedChan        chan bool
}

func newFlusher(exporterAddress string) flusher {
	return flusher{
		exporterAddress:       exporterAddress,
		maxExpireTimeMap:      make(map[int]expireTimeList),
		maxExpireTimeMapMutex: new(sync.RWMutex),
		minChangedChan:        make(chan bool, 1),
	}
}

func (f flusher) appendMaxExpireTime(stitcherID int, maxExpireTime uint64) {
	//this function should only be called from a single thread
	f.maxExpireTimeMapMutex.RLock()
	clockList, ok := f.maxExpireTimeMap[stitcherID]
	f.maxExpireTimeMapMutex.RUnlock()

	if !ok {
		clockList = expireTimeList{
			mutex:       new(sync.Mutex),
			expireTimes: list.New(),
		}
		clockList.expireTimes.PushBack(maxExpireTime)

		f.maxExpireTimeMapMutex.Lock()
		f.maxExpireTimeMap[stitcherID] = clockList
		f.maxExpireTimeMapMutex.Unlock()
		return
	}

	//NEED MUTEX?
	//Pushing onto an existing list will never effect findMinMaxExpireTime
	//since the clock lists are always required to have one element if
	//they exist.

	//May interact with stitcherDone? Mutex to be safe...
	clockList.mutex.Lock()
	clockList.expireTimes.PushBack(maxExpireTime)
	clockList.mutex.Unlock()
}

func (f flusher) stitcherDone(stitcherID int) {
	f.maxExpireTimeMapMutex.RLock()
	clockList := f.maxExpireTimeMap[stitcherID]
	f.maxExpireTimeMapMutex.RUnlock()

	//TODO: determine whether the list mutex is necessary
	clockList.mutex.Lock()
	defer clockList.mutex.Unlock()

	if clockList.expireTimes.Len() == 1 {
		return
	}

	clockList.expireTimes.Remove(clockList.expireTimes.Front())

	//Tell the flusher thread to look for a new minimum max expire time
	//since this stitcher may have held the last minimum max expire time
	//
	//We use a buffered channel so if a sticher changes the min
	//while the flusher is working, the flusher still knows
	//the min may have changed
	select {
	//the flusher is blocked
	case f.minChangedChan <- true:
	//the flusher is still flushing old records
	default:
	}
}

func (f flusher) findMinMaxExpireTime() uint64 {
	f.maxExpireTimeMapMutex.RLock()
	defer f.maxExpireTimeMapMutex.RUnlock()

	firstIter := true
	var minMaxExpireTime uint64

	for _, clockList := range f.maxExpireTimeMap {
		clockList.mutex.Lock()
		front := clockList.expireTimes.Front()
		if front != nil {
			maxExpireTimeForStitcher := front.Value.(uint64)
			if firstIter {
				minMaxExpireTime = maxExpireTimeForStitcher
				firstIter = false
			} else if maxExpireTimeForStitcher < minMaxExpireTime {
				minMaxExpireTime = maxExpireTimeForStitcher
			}
		}
		clockList.mutex.Unlock()
	}
	return minMaxExpireTime
}

func (f flusher) run(ctx context.Context, flusherDone *sync.WaitGroup,
	sessionsColl *mgo.Collection, sessionsOut chan<- *session.Aggregate) {

	var lastMinMaxExpireTime uint64 = math.MaxUint64

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case <-f.minChangedChan:
			minMaxExpireTime := f.findMinMaxExpireTime()
			//cache the minMaxExpireTime so we don't make extra db calls
			if lastMinMaxExpireTime == minMaxExpireTime {
				continue
			}

			//TODO: ensure err is not found error
			err := f.flushSession(minMaxExpireTime, sessionsColl, sessionsOut)
			for err == nil && ctx.Err() == nil {
				err = f.flushSession(minMaxExpireTime, sessionsColl, sessionsOut)
			}

			lastMinMaxExpireTime = minMaxExpireTime

			//Exit as soon as possible if the cancel signal comes through
			if ctx.Err() != nil {
				break Loop
			}

			//wait for stitcherDone() This prevents thrashing the mutexes
		}
	}

	err := f.flushSession(uint64(math.MaxInt64), sessionsColl, sessionsOut)
	for err == nil {
		err = f.flushSession(uint64(math.MaxInt64), sessionsColl, sessionsOut)
	}
	sessionsColl.Database.Session.Close()
	flusherDone.Done()
}

func (f flusher) flushSession(maxExpireTime uint64, sessionsColl *mgo.Collection, sessionsOut chan<- *session.Aggregate) error {
	var oldSession session.Aggregate
	_, err := sessionsColl.Find(bson.M{
		"flowEndMillisecondsAB": bson.M{
			"$lt": maxExpireTime,
		},
		"flowEndMillisecondsBA": bson.M{
			"$lt": maxExpireTime,
		},
		"exporter": f.exporterAddress,
	}).Apply(mgo.Change{
		Remove: true,
	}, &oldSession)

	if err != nil {
		return err
	}

	sessionsOut <- &oldSession
	return nil
}
