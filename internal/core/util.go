package core

import (
	"strconv"
	"strings"
)

// FormatFloat formats a float64 with appropriate precision
// Returns a string representation of the float
func FormatFloat(value float64, precision int) string {
	return strconv.FormatFloat(value, 'f', precision, 64)
}

// ParseFloat parses a string into a float64
// Returns the float value and any error encountered
func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// FormatWithOptimalPrecision formats a float using its inherent precision
// It determines the number of decimal places automatically
func FormatWithOptimalPrecision(value float64) string {
	precision := int(NumDecPlaces(value))
	return FormatFloat(value, precision)
}

// TrimTrailingZeros removes unnecessary zeros after the decimal point
func TrimTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}

	s = strings.TrimRight(s, "0")
	if s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}

	return s
}
