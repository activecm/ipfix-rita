package stitching

import (
	"math"
	"sync"
)

//MinMap is used to maintain a expiration clock
//for each exporting device.
//
//When a stitcher takes on a new flow, the last possible FlowEnd
//time which could correlate with the new flow (FlowStart - Threshold)
//is mapped into the MinMap with a stitcher id.
//
//When a stitcher finishes processing a flow, the stitcher
//checks if it is processing the flow with the minimum
//correlation time (FlowStart - Threshold).
//
//If the stitcher has the minimum correlation time,
//the stitcher removes all of the session aggregates with
//FlowEnd time < the correlation time and writes out the records. (Expiration)
//
//It is safe to assume the clock values stored will never be 0.
//0 may be used as a flag value.
//
//MinMap must be thread safe
type MinMap interface {
	//FindMinID returns the stitcherID with the minimum clock
	//across all stitchers. Returns -1 if empty.
	FindMinID() int
	//Set sets the clock value for a given stitcher
	Set(stitcherID int, clock uint64)
	//Clear clears the clock value for a given stitcher
	Clear(stitcherID int)
}

type sliceMinMap struct {
	clocks []uint64
	mutex  *sync.RWMutex
}

//NewSliceMinMap returns a new MinMap backed by a slice. Good
//for a small number of stitchers.
func NewSliceMinMap(maxStitcherID int) MinMap {
	return sliceMinMap{
		clocks: make([]uint64, maxStitcherID),
		mutex:  new(sync.RWMutex),
	}
}

//FindMinID returns the stitcherID with the minimum clock
//across all stitchers. Returns -1 if empty.
func (s sliceMinMap) FindMinID() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var min uint64 = math.MaxUint64
	var stitcherID = -1
	for i := range s.clocks {
		if s.clocks[i] != 0 && s.clocks[i] < min {
			min = s.clocks[i]
			stitcherID = i
		}
	}
	return stitcherID
}

//Set sets the clock value for a given stitcher
func (s sliceMinMap) Set(stitcherID int, clock uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clocks[stitcherID] = clock
}

//Clear clears the clock value for a given stitcher
func (s sliceMinMap) Clear(stitcherID int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clocks[stitcherID] = 0
}
