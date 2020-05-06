package monotime

import (
	"fmt"
	"os"
	"sync"
	"unsafe"
)

// MonoDuration is a monotonic interval of time in nanoseconds.
type MonoDuration int64

// Round returns the result of rounding d to the nearest multiple of m. The
// rounding behavior for halfway values is to round way from zero.
func (d MonoDuration) Round(m MonoDuration) MonoDuration {
	// Since they're actually the same type we get to cheat.
	// I think the compiler makes these conversions free?
	return MonoDuration(MonoTime(d).Round(m))
}

// Truncate returns the result of rounding d toward zero to a multiple of m.
// if m <= 0, Truncate returns d unchanged.
func (d MonoDuration) Truncate(m MonoDuration) MonoDuration {
	// Since they're actually the same type we get to cheat.
	// I think the compiler makes these conversions free?
	return MonoDuration(MonoTime(d).Truncate(m))
}

// MonoTime is a monotonic timestamp, measured as nanoseconds since some
// arbitrary time chosen by the system at boot.
type MonoTime int64

// Add returns the monotonic time t+d.
func (t MonoTime) Add(d MonoDuration) MonoTime {
	return t + MonoTime(d)
}

// Sub returns the monotonic duration t-u. To compute t-d for a duration d,
// use t.Add(-d).
func (t MonoTime) Sub(tt MonoTime) MonoDuration {
	return MonoDuration(t - tt)
}

// Round returns the result of routing t to the nearest multiple of d (since
// the zero time). The rounding behavior for halfway values is to round up. If
// d <= 0, Round returns t unchanged.
//
// Round operates on the time as an absolute duration since the zero time; it
// does not operate on the presentation form of the time. Thus, Round(Hour)
// may return a time with a non-zero minute, depending on the zero time of
// your system.
func (t MonoTime) Round(d MonoDuration) MonoTime {
	if d <= 0 {
		return t
	}

	r := MonoDuration(t) % d
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
func (t MonoTime) Truncate(d MonoDuration) MonoTime {
	if d <= 0 {
		return t
	}
	return t.Add(-(MonoDuration(t) % d))
}

// Ticker mimics time.Ticker, but uses a monotonic kernel timer.
type Ticker struct {
	C        <-chan struct{}
	stopOnce sync.Once
	stopped  bool
	fd       os.File
}

// Stop turns off a ticker. After Stop, no more ticks will be sent. Stop does
// not close the Channel, to prevent a concurrent goroutine reading from the
// channel from seeing an erroneous "tick".
func (t *Ticker) Stop() {
	t.stopOnce.Do(func() {
		t.fd.Close()
	})
}

func (t *Ticker) start() {
	expirations := make([]byte, 8)
	ch := make(chan struct{})
	t.C = ch
	go func() {
		for {
			_, err := t.fd.Read(expirations)
			if err == os.ErrClosed {
				return
			} else if err != nil {
				err = fmt.Errorf("Unknown error reading from timerfd: %w", err)
				panic(err)
			}
			// actively want to depend on host byte order.
			occurances := *(*uint64)(unsafe.Pointer(&expirations[0]))
			var i uint64
			for i = 0; i < occurances; i++ {
				ch <- struct{}{}
			}
		}
	}()
}

func (t *Ticker) Reset(d MonoDuration) bool {
	return t.ResetAt(-1, d)
}
func (t *Ticker) ResetAt(tt MonoTime, d MonoDuration) bool

func NewTicker(d MonoDuration) *Ticker {
	return NewTickerAt(-1, d)
}

func NewTickerAt(t MonoTime, d MonoDuration) *Ticker
