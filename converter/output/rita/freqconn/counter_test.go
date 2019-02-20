package freqconn_test

import (
	"errors"
	"testing"

	"github.com/activecm/ipfix-rita/converter/output/rita/freqconn"
	"github.com/stretchr/testify/require"
)

//TestThresholdMet ensures thresholdMet is called
//when the connection counter hits the threshold but
//thresholdExceeded is not
func TestThresholdMet(t *testing.T) {
	shouldPass := false
	thresholdMet := func(conn freqconn.UConnPair, count int) error {
		shouldPass = true
		return nil
	}
	thresholdExceeded := func(conn freqconn.UConnPair, count int) error {
		t.Fatalf("thresholdExceeded called when it should not have been. count %d, threshold %d", count, testThreshold)
		return nil
	}
	c := freqconn.NewConnCounter(testThreshold, thresholdMet, thresholdExceeded)

	testConnection := freqconn.UConnPair{
		Src: "1.1.1.1",
		Dst: "2.2.2.2",
	}

	for i := 0; i < testThreshold-1; i++ {
		funcRan, err := c.Increment(testConnection)
		require.False(t, funcRan, "Increment said a threshold function ran when it should not have")
		require.Nil(t, err, "Increment returned an error when it shouldn't have")
	}

	funcRan, err := c.Increment(testConnection)
	require.True(t, funcRan, "Increment said threshold function did not run when it should have")
	require.Nil(t, err, "Increment returned an error when it shouldn't have")

	require.True(t, shouldPass, "thresholdMet was not called.")
}

//TestThresholdExceeded ensures thresholdExceeded is called
//when the connection counter exceeds the threshold but
//thresholdExceeded is not
func TestThresholdExceeded(t *testing.T) {
	shouldPass := false
	thresholdMetCalledOnce := false
	thresholdMet := func(conn freqconn.UConnPair, count int) error {
		if thresholdMetCalledOnce {
			t.Fatalf("thresholdMet called when it should not have been. count %d, threshold %d", count, testThreshold)
		} else {
			thresholdMetCalledOnce = true
		}
		return nil
	}
	thresholdExceeded := func(conn freqconn.UConnPair, count int) error {
		shouldPass = true
		return nil
	}
	c := freqconn.NewConnCounter(testThreshold, thresholdMet, thresholdExceeded)

	testConnection := freqconn.UConnPair{
		Src: "1.1.1.1",
		Dst: "2.2.2.2",
	}

	for i := 0; i < testThreshold-1; i++ {
		funcRan, err := c.Increment(testConnection)
		require.False(t, funcRan, "Increment said a threshold function ran when it should not have")
		require.Nil(t, err, "Increment returned an error when it shouldn't have")
	}

	funcRan, err := c.Increment(testConnection)
	require.True(t, funcRan, "Increment said threshold function did not run when it should have")
	require.Nil(t, err, "Increment returned an error when it shouldn't have")

	funcRan, err = c.Increment(testConnection)
	require.True(t, funcRan, "Increment said threshold function did not run when it should have")
	require.Nil(t, err, "Increment returned an error when it shouldn't have")

	require.True(t, shouldPass, "thresholdExceeded was not called.")
}

//TestErrorsReturned ensures the errors returned from thresholdExceeded
//and thresholdMet are returned via Increment. Additionally the test
//asserts that the counter should not be incremented if there is an error.
func TestErrorsReturned(t *testing.T) {
	thresholdMetErr := errors.New("thresholdMet error")
	thresholdExceededErr := errors.New("thresholdExceeded error")

	thresholdMet := func(conn freqconn.UConnPair, count int) error {
		return thresholdMetErr
	}
	thresholdExceeded := func(conn freqconn.UConnPair, count int) error {
		return thresholdExceededErr
	}

	c := freqconn.NewConnCounter(1, thresholdMet, thresholdExceeded)

	testConnection := freqconn.UConnPair{
		Src: "1.1.1.1",
		Dst: "2.2.2.2",
	}

	funcRan, err := c.Increment(testConnection)
	require.True(t, funcRan, "Increment said threshold function did not run when it should have")
	require.Equal(t, thresholdMetErr, err, "error from thresholdMet not returned")

	funcRan, err = c.Increment(testConnection)
	require.True(t, funcRan, "Increment said threshold function did not run when it should have")
	require.Equal(t, thresholdMetErr, err, "error from thresholdMet not returned")

	c = freqconn.NewConnCounter(1, func(freqconn.UConnPair, int) error { return nil }, thresholdExceeded)
	c.Increment(testConnection)
	funcRan, err = c.Increment(testConnection)
	require.True(t, funcRan, "Increment said threshold function did not run when it should have")
	require.Equal(t, thresholdExceededErr, err, "error from thresholdMet not returned")
}
