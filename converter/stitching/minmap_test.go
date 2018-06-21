package stitching

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSliceMinMap(t *testing.T) {
	minMap := NewSliceMinMap(10)
	require.Equal(t, -1, minMap.FindMinID())

	var i int
	for ; i < 10; i++ {
		minMap.Set(i, uint64(100*(i+1)))
	}
	require.Equal(t, 0, minMap.FindMinID())

	for i = 0; i < 8; i++ {
		minMap.Clear(i)
	}
	require.Equal(t, 8, minMap.FindMinID())

	minMap.Clear(8)
	require.Equal(t, 9, minMap.FindMinID())

	minMap.Clear(9)
	require.Equal(t, -1, minMap.FindMinID())
}
