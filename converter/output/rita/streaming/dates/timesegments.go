package dates

import (
	"time"
)

//SegmentRelativeTimestampFactory translates unix timestamps in milliseconds
//into segment relative timestamps. Segment relative timestamps represent
//timestamps as offsets from evenly divided time segments.
//The timezone determines the alignment of the time segments.
//For instance, daily segments can be configured with a segmentDurationMillis
//of 86400 * 1000. If the timezone is set to UTC, then the segments will
//align with the unix timestamps. However, you likely want to align segments
//to local time. Setting the timezone shifts each of the segments to
//the left or right by a constant (the timezone offset from UTC).
type SegmentRelativeTimestampFactory struct {
	segmentDurationMillis int64
	timezoneOffsetMillis  int64
}

//NewSegmentRelativeTimestampFactory creates a new SegmentRelativeTimestampFactory
//which creates SegmentRelativeTimestamps which are represented as offsets
//from evenly divided time segments as determined by segmentDurationMillis.
func NewSegmentRelativeTimestampFactory(segmentDurationMillis int64, timezone *time.Location) SegmentRelativeTimestampFactory {
	_, timezoneOffsetSeconds := time.Now().In(timezone).Zone()
	return SegmentRelativeTimestampFactory{
		segmentDurationMillis: segmentDurationMillis,
		timezoneOffsetMillis:  int64(timezoneOffsetSeconds) * 1000,
	}
}

//GetSegmentRelativeTimestamp creates a new SegmentRelativeTimestamp
//from a unix timestamp given in milliseconds relative to the segment length
//determined during the construction of the SegmentRelativeTimestampFactory.
func (t SegmentRelativeTimestampFactory) GetSegmentRelativeTimestamp(unixTSMillis int64) SegmentRelativeTimestamp {
	s := SegmentRelativeTimestamp{
		SegmentStartMillis:           ((unixTSMillis+t.timezoneOffsetMillis)/t.segmentDurationMillis)*t.segmentDurationMillis - t.timezoneOffsetMillis,
		SegmentDurationMillis:        t.segmentDurationMillis,
		OffsetFromSegmentStartMillis: (unixTSMillis + t.timezoneOffsetMillis) % t.segmentDurationMillis,
	}
	//handle times before the unix epoch
	if s.OffsetFromSegmentStartMillis < 0 {
		s.SegmentStartMillis -= t.segmentDurationMillis
		s.OffsetFromSegmentStartMillis = t.segmentDurationMillis + s.OffsetFromSegmentStartMillis
	}
	return s
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
