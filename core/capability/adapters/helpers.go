package adapters

import (
	"errors"
	"time"
)

// ErrNotImplemented is returned when a method is not implemented by the underlying provider.
var ErrNotImplemented = errors.New("method not implemented by underlying provider")

// timeFromUnix converts a Unix timestamp (seconds) to time.Time.
func timeFromUnix(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// timeToUnix converts time.Time to Unix timestamp (seconds).
func timeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
