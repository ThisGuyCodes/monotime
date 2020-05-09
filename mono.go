package monotime

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// Time is a monotonic timestamp, measured as nanoseconds since some
// arbitrary time chosen by the system at boot.
type Time int64

// Add returns the monotonic time t+d.
func (t Time) Add(d time.Duration) Time {
	return t + Time(d)
}

// Sub returns the monotonic duration t-u. To compute t-d for a duration d,
// use t.Add(-d).
func (t Time) Sub(tt Time) time.Duration {
	return time.Duration(t - tt)
}

// Round returns the result of routing t to the nearest multiple of d (since
// the zero time). The rounding behavior for halfway values is to round up. If
// d <= 0, Round returns t unchanged.
//
// Round operates on the time as an absolute duration since the zero time; it
// does not operate on the presentation form of the time. Thus, Round(Hour)
// may return a time with a non-zero minute, depending on the zero time of
// your system.
func (t Time) Round(d time.Duration) Time {
	if d <= 0 {
		return t
	}

	r := time.Duration(t) % d
	if r*2 <= d {
		return t.Add(-r)
	}
	return t.Add(d - r)
}

// Truncate returns the result of rounding t down to a multiple of d (since
// the zero time). if d <= 0, Truncate returns t unchanged.
//
// Truncate operates on the time as an absolute duration since the zero time;
// it does not operate on the presentation form of the time. Thus,
// Truncate(Hour) may return a time with a non-zero minute, depending on the
// zero time of your system.
func (t Time) Truncate(d time.Duration) Time {
	if d <= 0 {
		return t
	}
	return t.Add(-(time.Duration(t) % d))
}

// Now gets the current monotonic time
//
// Monotonic time is *not comparable* accross sytems, or even reboots.
func Now() Time {
	spec := new(unix.Timespec)
	err := unix.ClockGettime(unix.CLOCK_MONOTONIC, spec)
	if err != nil {
		err = fmt.Errorf("Error getting monotime from the kernel: %w", err)
		panic(err)
	}
	return Time(spec.Nano())
}
