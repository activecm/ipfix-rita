package rammatch

import (
	"container/heap"
	"container/list"
	"sync"
	"sync/atomic"

	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/stitching/matching"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/pkg/errors"
)

//listSessionIterator wraps *list.List to provide the
//appropriate signature to implement session.Iterator
type listSessionIterator struct {
	data     *list.List
	iterNode *list.Element
}

func newListSessionIterator(data *list.List) session.Iterator {
	if data == nil {
		return &listSessionIterator{}
	}
	return &listSessionIterator{
		data:     data,
		iterNode: data.Front(),
	}
}

func (l *listSessionIterator) Next(sessAgg *session.Aggregate) bool {
	if l.data == nil || l.iterNode == nil {
		return false
	}

	*sessAgg = *(l.iterNode.Value.(*session.Aggregate))
	l.iterNode = l.iterNode.Next()
	return true
}

func (l *listSessionIterator) Err() error {
	return nil
}

//ramMatcher provides an implementation of Matcher entirely in RAM
type ramMatcher struct {
	matchMap      sync.Map
	insertTracker uint64
	count         uint64

	sessionsOut      chan<- *session.Aggregate
	preFlushMaxSize  uint64
	postFlushMaxSize uint64

	log logging.Logger
}

//NewRAMMatcher returns a new matcher which operates entirely in RAM
func NewRAMMatcher(log logging.Logger, sessionsOut chan<- *session.Aggregate,
	maxSize uint64, flushToPercent float32) matching.Matcher {
	return &ramMatcher{
		sessionsOut:      sessionsOut,
		preFlushMaxSize:  maxSize,
		postFlushMaxSize: uint64(float32(maxSize)*flushToPercent + 0.5),
		log:              log,
	}
}

//Close tears down any resources consumed by the Matcher
//and flushes any remaining Aggregates from the matcher.
func (r *ramMatcher) Close() error {
	return r.flushTo(0)
}

//Find searches the Matcher for Aggregates which
//match the given AggregateQuery
func (r *ramMatcher) Find(sessAggQuery *session.AggregateQuery) session.Iterator {
	resultsListIface, ok := r.matchMap.Load(*sessAggQuery)
	if !ok {
		return newListSessionIterator(nil)
	}
	return newListSessionIterator(resultsListIface.(*list.List))
}

//Insert adds a session aggregate to the Matcher.
//Insert is responsible for setting the Aggregate.MatcherID field.
//MatcherID must be used to disambiguate between aggregates in the
//matcher with the same session.AggregateQuery. Usually MatcherID
//is some sort of auto incrementing ID.
func (r *ramMatcher) Insert(sessAgg *session.Aggregate) error {
	sessAgg.MatcherID = atomic.AddUint64(&r.insertTracker, 1)
	newList := list.New()
	newList.PushBack(sessAgg)
	existingList, loaded := r.matchMap.LoadOrStore(sessAgg.AggregateQuery, newList)
	if loaded {
		existingList.(*list.List).PushBack(sessAgg)
	}
	atomic.AddUint64(&r.count, 1)
	return nil
}

//Update finds an Aggregate in the Matcher using the given
//Aggregate's AggregateQuery and MatcherID and updates
//the matching Aggregate's data.
func (r *ramMatcher) Update(sessAgg *session.Aggregate) error {
	resultsListIface, ok := r.matchMap.Load(sessAgg.AggregateQuery)
	if !ok {
		return errors.Errorf("no records found for AggregateQuery:\n%+v", sessAgg.AggregateQuery)
	}
	resultsList := resultsListIface.(*list.List)
	for iterNode := resultsList.Front(); iterNode != nil; iterNode = iterNode.Next() {
		otherSessAgg := iterNode.Value.(*session.Aggregate)
		if otherSessAgg.MatcherID == sessAgg.MatcherID {
			//copy the new data into the pointer
			*otherSessAgg = *sessAgg
			return nil
		}
	}
	return errors.Errorf("no records found for MatcherID: %d", sessAgg.MatcherID)
}

//Remove finds an Aggregate in the Matcher using the given
//Aggregate's AggregateQuery and AggregateID and removes it
//from the system.
func (r *ramMatcher) Remove(sessAgg *session.Aggregate) error {
	resultsListIface, ok := r.matchMap.Load(sessAgg.AggregateQuery)
	if !ok {
		return errors.Errorf("no records found for AggregateQuery:\n%+v", sessAgg.AggregateQuery)
	}
	resultsList := resultsListIface.(*list.List)
	for iterNode := resultsList.Front(); iterNode != nil; iterNode = iterNode.Next() {
		otherSessAgg := iterNode.Value.(*session.Aggregate)
		if otherSessAgg.MatcherID == sessAgg.MatcherID {
			//we don't have to worry about breaking the iteration with Remove
			//since we return immediately
			resultsList.Remove(iterNode)
			atomic.AddUint64(&r.count, ^uint64(0)) //-1 in two's complement >.>
			return nil
		}
	}
	return errors.Errorf("no records found for MatcherID: %d", sessAgg.MatcherID)
}

//ShouldFlush returns true if Flush should be called in order
//to maintain performance and ensure unmatched records are
//written out in a timely manner.
func (r *ramMatcher) ShouldFlush() (bool, error) {
	return atomic.LoadUint64(&r.count) > r.preFlushMaxSize, nil
}

//Flush evicts Aggregates from the Matcher in order to maintain
//performance and ensure unmatched records are written out in a
//timely manner.
func (r *ramMatcher) Flush() error {
	return r.flushTo(r.postFlushMaxSize)
}

func (r *ramMatcher) flushTo(targetCount uint64) error {
	startCount := atomic.LoadUint64(&r.count)
	if startCount <= targetCount {
		return nil
	}
	defer func() {
		r.log.Info("finished session aggregate flush", logging.Fields{
			"start count":   startCount,
			"current count": atomic.LoadUint64(&r.count),
			"target count":  targetCount,
		})
	}()
	//flush out the garbage first
	for i := int64(1); i <= 2; i++ {
		r.flushNPacketConnections(i)
		if atomic.LoadUint64(&r.count) <= targetCount {
			return nil
		}
	}
	r.flushOldest(targetCount)
	return nil
}

//flushNPacketConnections flushes sessions which contain
//exactly n packets in one direction and 0 in the other
func (r *ramMatcher) flushNPacketConnections(n int64) error {
	r.matchMap.Range(func(aggQueryIface interface{}, aggListIface interface{}) bool {
		aggList := aggListIface.(*list.List)

		//https://stackoverflow.com/questions/27662614/how-to-remove-element-from-list-while-iterating-the-same-list-in-golang
		var next *list.Element
		for iterNode := aggList.Front(); iterNode != nil; iterNode = next {
			next = iterNode.Next()
			sessAgg := iterNode.Value.(*session.Aggregate)

			if sessAgg.PacketTotalCountAB == n && sessAgg.PacketTotalCountBA == int64(0) ||
				sessAgg.PacketTotalCountBA == n && sessAgg.PacketTotalCountAB == int64(0) {

				//write out the session aggregate
				r.sessionsOut <- sessAgg

				aggList.Remove(iterNode)
				atomic.AddUint64(&r.count, ^uint64(0)) //-1 in two's complement >.>
			}
		}
		if aggList.Len() == 0 {
			r.matchMap.Delete(aggQueryIface.(session.AggregateQuery))
		}
		return true
	})
	return nil
}

//sessionAggregateHeap defines a heap used for flushing old records out of the
//ramMatcher map
type sessionAggregateHeap []*session.Aggregate

func (h sessionAggregateHeap) Len() int { return len(h) }
func (h sessionAggregateHeap) Less(i, j int) bool {
	return h[i].MatcherID.(uint64) < h[j].MatcherID.(uint64)
}
func (h sessionAggregateHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *sessionAggregateHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*session.Aggregate))
}

func (h *sessionAggregateHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (r *ramMatcher) flushOldest(targetCount uint64) error {
	minHeap := new(sessionAggregateHeap)
	heap.Init(minHeap)
	r.matchMap.Range(func(aggQueryIface interface{}, aggListIface interface{}) bool {
		aggList := aggListIface.(*list.List)

		for iterNode := aggList.Front(); iterNode != nil; iterNode = iterNode.Next() {
			heap.Push(minHeap, iterNode.Value.(*session.Aggregate))
		}
		return true
	})

	for atomic.LoadUint64(&r.count) > targetCount && len(*minHeap) > 0 {
		aggToRemove := heap.Pop(minHeap).(*session.Aggregate)

		r.sessionsOut <- aggToRemove
		r.Remove(aggToRemove)
	}
	return nil
}
