package devices

import (
	"strconv"
)

func Float64ToString(num float32) string {
	return strconv.FormatFloat(float64(num), 'f', -1, 32)
}

func Int64ToString(n int64) string {
	return strconv.FormatInt(int64(n), 10)
}


