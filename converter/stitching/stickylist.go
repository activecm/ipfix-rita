package stitching

import "container/list"

type stickySortedClockList struct {
	inner       list.List
	replaceList list.List
	stickyCount int
}

type stickySortedClockListElement struct {
	time             int64
	markedForRemoval *bool
}

func newStickySortedClockList(stickyCount int) *stickySortedClockList {
	sscl := &stickySortedClockList{
		inner:       list.List{},
		replaceList: list.List{},
		stickyCount: stickyCount,
	}
	sscl.inner.Init()
	sscl.replaceList.Init()
	return sscl
}

func (s *stickySortedClockList) len() int {
	return s.inner.Len()
}

func (s *stickySortedClockList) getMinimumTime() (int64, bool) {
	if s.inner.Len() == 0 {
		return 0, false
	}
	frontVal := s.inner.Front().Value.(stickySortedClockListElement)
	return frontVal.time, true
}

func (s *stickySortedClockList) addTime(time int64) {
	if s.replaceList.Len() != 0 {
		nextRemovalBookkeeperNode := s.replaceList.Front()
		nodeToBeRemoved := nextRemovalBookkeeperNode.Value.(*list.Element)
		s.inner.Remove(nodeToBeRemoved)
		s.replaceList.Remove(nextRemovalBookkeeperNode)
	}

	timeElement := stickySortedClockListElement{
		time:             time,
		markedForRemoval: new(bool),
	}

	//Do a sorted insert
	currNode := s.inner.Front()
	unbox := func(node *list.Element) int64 { return node.Value.(stickySortedClockListElement).time }

	if currNode == nil || unbox(currNode) >= time {
		//empty list or time is the new minimum
		s.inner.PushFront(timeElement)
	} else {
		//insert into the middle or end
		for currNode.Next() != nil && unbox(currNode.Next()) < time {
			currNode = currNode.Next()
		}
		s.inner.InsertAfter(timeElement, currNode)
	}
}

func (s *stickySortedClockList) removeTime(time int64) bool {

	matching := func(node *list.Element, time int64) bool {
		elem := node.Value.(stickySortedClockListElement)
		return elem.time == time && !*elem.markedForRemoval
	}

	currNode := s.inner.Front()

	for !matching(currNode, time) && currNode != nil {
		currNode = currNode.Next()
	}

	if currNode == nil {
		return false
	}

	if s.inner.Len() <= s.stickyCount {
		//ensure we don't queue up a node for removal twice
		//If the list has duplicates, the search above
		//will always return the first element matching the given
		//time if don't add a marker.
		elem := currNode.Value.(stickySortedClockListElement)
		*elem.markedForRemoval = true

		s.replaceList.PushBack(currNode)
	} else {
		s.inner.Remove(currNode)
	}

	return true
}
