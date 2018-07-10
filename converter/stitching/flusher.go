package stitching

import (
	"context"
	"math"
	"sync"

	"github.com/activecm/ipfix-rita/converter/stitching/session"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type flusher struct {
	exporterAddress  string
	maxExpireTimeMap map[int]*stickySortedClockList
	mutex            *sync.RWMutex
	minChangedChan   chan bool
}

func newFlusher(exporterAddress string) flusher {
	return flusher{
		exporterAddress:  exporterAddress,
		maxExpireTimeMap: make(map[int]*stickySortedClockList),
		mutex:            new(sync.RWMutex),
		minChangedChan:   make(chan bool, 1),
	}
}

func (f flusher) addMaxExpireTime(stitcherID int, maxExpireTime int64) {
	//this function should only be called from a single thread
	f.mutex.Lock()
	defer f.mutex.Unlock()

	/*
		Debug code
		var sortedById [5]int

		for sId := range f.maxExpireTimeMap {
			sortedById[sId] = f.maxExpireTimeMap[sId].len()
		}
		for sId := range f.maxExpireTimeMap {
			fmt.Printf("%d, ", sortedById[sId])
		}
		f.mutex.Unlock()
		fmt.Printf("%d\n", f.findMinMaxExpireTime())

		f.mutex.Lock()
		defer f.mutex.Unlock()
	*/
	clockList, ok := f.maxExpireTimeMap[stitcherID]

	//first element
	if !ok {
		clockList = newStickySortedClockList(10)
		f.maxExpireTimeMap[stitcherID] = clockList
	}

	//Do a sorted insert, possibly evicting a stickied record
	clockList.addTime(maxExpireTime)

	//Tell the flusher thread to look for a new minimum max expire time
	//since this maxExpireTime may be the new minimum
	//
	//We use a buffered channel so if a the min changes
	//while the flusher is working, the flusher still knows
	//the min may have changed
	select {
	//the flusher is blocked
	case f.minChangedChan <- true:
	//the flusher is still flushing old records
	default:
	}
}

func (f flusher) removeMaxExpireTime(stitcherID int, maxExpireTime int64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	//get the stitcher specific list
	clockList := f.maxExpireTimeMap[stitcherID]

	//remove the maxExpireTime from the stitcher specific list
	//this may not truely remove the time from the list if
	//there are too few items in the list. Instead, it will be scheduled
	//for eviction
	clockList.removeTime(maxExpireTime)

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

func (f flusher) findMinMaxExpireTime() int64 {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	firstIter := true
	var minMaxExpireTime int64

	for _, clockList := range f.maxExpireTimeMap {
		minMaxExpireTimeForStitcher, ok := clockList.getMinimumTime()
		if ok {
			if firstIter {
				minMaxExpireTime = minMaxExpireTimeForStitcher
				firstIter = false
			} else if minMaxExpireTimeForStitcher < minMaxExpireTime {
				minMaxExpireTime = minMaxExpireTimeForStitcher
			}
		}
	}
	return minMaxExpireTime
}

func (f flusher) run(ctx context.Context, flusherDone *sync.WaitGroup,
	sessionsColl *mgo.Collection, sessionsOut chan<- *session.Aggregate) {

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case <-f.minChangedChan:
			/*
				The flusher is horribly broken.
				minMaxExpireTime := f.findMinMaxExpireTime()

				//Keep flushing until there is nothing left to flush
				var err error
				for err != mgo.ErrNotFound {

					//update minMaxExpireTime if it changes while we are flushing
					select {
					case <-f.minChangedChan:
						minMaxExpireTime = f.findMinMaxExpireTime()
					case <-ctx.Done():
						break Loop
					default:
					}

					err = f.flushSession(minMaxExpireTime, sessionsColl, sessionsOut)
					//TODO: Hanle errors that are not mgo.ErrNotFound
				}
				//wait for addMaxExpireTime()/ removeMaxExpireTime() This prevents thrashing the mutexes
			*/
		}
	}

	err := f.flushSession(math.MaxInt64, sessionsColl, sessionsOut)
	for err == nil {
		err = f.flushSession(math.MaxInt64, sessionsColl, sessionsOut)
	}
	sessionsColl.Database.Session.Close()
	flusherDone.Done()
}

func (f flusher) flushSession(maxExpireTime int64, sessionsColl *mgo.Collection, sessionsOut chan<- *session.Aggregate) error {
	//fmt.Printf("Flushing: %d\n", maxExpireTime)
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
