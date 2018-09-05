package dates

import (
	"time"
)

//SegmentRelativeTimestampFactory translates unix timestamps in milliseconds
//into segment relative timestamps. Segment relative timestamps represent
//timestamps as offsets from evenly divided time segments.
type SegmentRelativeTimestampFactory struct {
	segmentDurationMillis int64
}

//NewSegmentRelativeTimestampFactory creates a new SegmentRelativeTimestampFactory
//which creates SegmentRelativeTimestamps which are represented as offsets
//from evenly divided time segments as determined by segmentDurationMillis.
func NewSegmentRelativeTimestampFactory(segmentDurationMillis int64) SegmentRelativeTimestampFactory {
	return SegmentRelativeTimestampFactory{
		segmentDurationMillis: segmentDurationMillis,
	}
}

//GetSegmentRelativeTimestamp creates a new SegmentRelativeTimestamp
//from a unix timestamp given in milliseconds relative to the segment length
//determined during the construction of the SegmentRelativeTimestampFactory.
func (t SegmentRelativeTimestampFactory) GetSegmentRelativeTimestamp(unixTSMillis int64) SegmentRelativeTimestamp {
	return SegmentRelativeTimestamp{
		SegmentStartMillis:           (unixTSMillis / t.segmentDurationMillis) * t.segmentDurationMillis,
		SegmentDurationMillis:        t.segmentDurationMillis,
		OffsetFromSegmentStartMillis: unixTSMillis % t.segmentDurationMillis,
	}
}

//Now creates a new SegmentRelativeTimestamp for the current local time
//given in milliseconds relative to the segment length
//determined during the construction of the SegmentRelativeTimestampFactory.
func (t SegmentRelativeTimestampFactory) Now() SegmentRelativeTimestamp {
	return t.GetSegmentRelativeTimestamp(time.Now().Unix() * 1000)
}

//SegmentRelativeTimestamp represents a timestamp relative to an interval of time.
type SegmentRelativeTimestamp struct {
	SegmentStartMillis           int64
	SegmentDurationMillis        int64
	OffsetFromSegmentStartMillis int64
}

//SegmentOffsetFrom computes how many segments this segment is away from another
//segment. If the segments are different lengths (durations), the function
//returns (0, false). Otherwise, the function returns x such that
//this.SegmentStartMillis + x * this.SegmentDurationMillis = s2.SegmentStartMillis
//and true.
func (s SegmentRelativeTimestamp) SegmentOffsetFrom(s2 SegmentRelativeTimestamp) (int64, bool) {
	if s.SegmentDurationMillis != s2.SegmentDurationMillis {
		return 0, false
	}
	return (s2.SegmentStartMillis - s.SegmentStartMillis) / s.SegmentDurationMillis, true
}
