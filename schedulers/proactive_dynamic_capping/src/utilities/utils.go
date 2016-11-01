package utils

import "strconv"

// Convert float64 to string
func FloatToString(input float64) string {
  // Precision is 2, Base is 64
  return strconv.FormatFloat(input, 'f', 2, 64)
}
