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
// the ThresholdMet method on the given ConnCountNotifier will be executed
// with the UConnPair and the new count. If the count then exceeds the threshold,
// the ThresholdExceeded method will then be ran in a similar fashion.
type ConnCounter struct {
	connectionCounts map[UConnPair]int
	threshold        int
	notifier         ConnCountNotifier
}

// ConnCountNotifier specifies an interface for updating an external component
// with the new count for a given connection pair. ThresholdMet will be called
// when the count hits a specified threshold, and ThresholdExceeded will be
// called when the count exceeds a specified threshold.
type ConnCountNotifier interface {
	ThresholdMet(UConnPair, int) error
	ThresholdExceeded(UConnPair, int) error
}

// NewConnCounter creates a new ConnCounter. Each unique connection
// starts at 0.
func NewConnCounter(threshold int, notifier ConnCountNotifier) ConnCounter {
	return ConnCounter{
		connectionCounts: make(map[UConnPair]int),
		threshold:        threshold,
		notifier:         notifier,
	}
}

// NewConnCounterFromMap creates a new ConnCounter. Each unique
// connection starts with the counts supplied in the data map.
func NewConnCounterFromMap(data map[UConnPair]int, threshold int, notifier ConnCountNotifier) ConnCounter {
	c := ConnCounter{
		connectionCounts: data,
		threshold:        threshold,
		notifier:         notifier,
	}
	return c
}

// Increment increments the count corresponding to the
// UConnPair passed in. If the ConnCounter threshold is
// met, thresholdMetFunc is ran. Alternatively, if the
// threshold is exceeded, thresholdExceededFunc is ran.
// Returns true if either thresholdMet or thresholdExceeded
// is called. May return an error from either function.
// If an error is returned, the count is not updated.
func (f ConnCounter) Increment(connectionPair UConnPair) (bool, error) {
	newCount := f.connectionCounts[connectionPair] + 1
	var err error
	funcRan := false
	if newCount == f.threshold {
		err = f.notifier.ThresholdMet(connectionPair, newCount)
		funcRan = true
	} else if newCount > f.threshold {
		err = f.notifier.ThresholdExceeded(connectionPair, newCount)
		funcRan = true
	}
	if err == nil {
		f.connectionCounts[connectionPair] = newCount
	}
	return funcRan, err
}
