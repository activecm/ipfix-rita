package freqconn

// UConnPair records a unique connection pair. i.e.
// two ip addresses. Used to track how many times
// two hosts talk to each other
type UConnPair struct {
	Src string `bson:"src"`
	Dst string `bson:"dst"`
}

// FreqConn records how many times a unique connection pair
// connected
type FreqConn struct {
	UConnPair       `bson:",inline"`
	ConnectionCount int `bson:"connection_count"`
}

// ConnCounter tracks how many UConnPairs with
// matching source and destination addresses have been processed.
// When the count for a given UConnPair meets the threshold,
// the given thresholdMetFunc will be executed with the UConnPair and the
// new count. If the count then exceeds the threshold,
// the given thresholdExceededFunc will then be ran in a similar fashion.
type ConnCounter struct {
	connectionCounts      map[UConnPair]int
	threshold             int
	thresholdMetFunc      func(UConnPair, int) error
	thresholdExceededFunc func(UConnPair, int) error
}

// NewConnCounter creates a new ConnCounter. Each unique connection
// starts at 0.
func NewConnCounter(threshold int,
	thresholdMetFunc, thresholdExceededFunc func(UConnPair, int) error) ConnCounter {
	return ConnCounter{
		connectionCounts:      make(map[UConnPair]int),
		threshold:             threshold,
		thresholdMetFunc:      thresholdMetFunc,
		thresholdExceededFunc: thresholdExceededFunc,
	}
}

// NewConnCounterFromArray creates a new ConnCounter. Each unique
// connection starts with the counts supplied in the FreqConn array.
func NewConnCounterFromArray(data []FreqConn, threshold int,
	thresholdMetFunc, thresholdExceededFunc func(UConnPair, int) error) ConnCounter {
	c := ConnCounter{
		connectionCounts:      make(map[UConnPair]int),
		threshold:             threshold,
		thresholdMetFunc:      thresholdMetFunc,
		thresholdExceededFunc: thresholdExceededFunc,
	}
	for _, freqConn := range data {
		c.connectionCounts[freqConn.UConnPair] = freqConn.ConnectionCount
	}
	return c
}

// Increment increments the count corresponding to the
// UConnPair passed in. If the ConnCounter threshold is
// met, thresholdMetFunc is ran. Alternatively, if the
// threshold is exceeded, thresholdExceededFunc is ran.
func (f ConnCounter) Increment(connectionPair UConnPair) (bool, error) {
	newCount := f.connectionCounts[connectionPair] + 1
	f.connectionCounts[connectionPair] = newCount
	if newCount == f.threshold {
		err := f.thresholdMetFunc(connectionPair, newCount)
		return true, err
	} else if newCount > f.threshold {
		err := f.thresholdExceededFunc(connectionPair, newCount)
		return true, err
	}
	return false, nil
}
