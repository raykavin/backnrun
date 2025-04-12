package core

import (
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

// Series is a time series of ordered values
// It provides methods for analyzing time series data
type Series[T constraints.Ordered] []T

// Values returns the underlying slice of values
func (s Series[T]) Values() []T {
	return s
}

// Length returns the number of values in the series
func (s Series[T]) Length() int {
	return len(s)
}

// Last returns the value at a specified position from the end
// position 0 is the last value, 1 is the second-to-last, etc.
func (s Series[T]) Last(position int) T {
	return s[len(s)-1-position]
}

// LastValues returns a slice with the last 'size' values
// If size exceeds the length, returns the entire series
func (s Series[T]) LastValues(size int) Series[T] {
	if l := len(s); l > size {
		return s[l-size:]
	}
	return s
}

// Crossover detects when this series crosses above the reference series
// Returns true when the current value is higher, but the previous value was not
func (s Series[T]) Crossover(ref Series[T]) bool {
	return s.Last(0) > ref.Last(0) && s.Last(1) <= ref.Last(1)
}

// Crossunder detects when this series crosses below the reference series
// Returns true when the current value is lower/equal, but the previous value was higher
func (s Series[T]) Crossunder(ref Series[T]) bool {
	return s.Last(0) <= ref.Last(0) && s.Last(1) > ref.Last(1)
}

// Cross detects when this series crosses the reference series in either direction
func (s Series[T]) Cross(ref Series[T]) bool {
	return s.Crossover(ref) || s.Crossunder(ref)
}

// NumDecPlaces returns the number of decimal places in a float64
// Useful for formatting with appropriate precision
func NumDecPlaces(v float64) int64 {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return int64(len(s) - i - 1)
	}
	return 0
}
