package agent

import "time"

func getCurrentTimeMs() int64 {
	// Use time package to get current time in milliseconds
	return time.Now().UnixMilli()
}
