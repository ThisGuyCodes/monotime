package monotime

import (
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"
)

// MonoTime is a monotonic timestamp, measured as nanoseconds since some
// arbitrary time chosen by the system at boot.
type MonoTime int64

// Add returns the monotonic time t+d.
func (t MonoTime) Add(d time.Duration) MonoTime {
	return t + MonoTime(d)
}

// Sub returns the monotonic duration t-u. To compute t-d for a duration d,
// use t.Add(-d).
func (t MonoTime) Sub(tt MonoTime) time.Duration {
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
func (t MonoTime) Round(d time.Duration) MonoTime {
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
func (t MonoTime) Truncate(d time.Duration) MonoTime {
	if d <= 0 {
		return t
	}
	return t.Add(-(time.Duration(t) % d))
}

// Ticker mimics time.Ticker, but uses a monotonic kernel timer.
type Ticker struct {
	C        <-chan struct{}
	stopOnce sync.Once
	stopped  bool
	fd       os.File
	m        sync.Mutex
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
		t.m.Lock()
		defer t.m.Unlock()
		for {
			_, err := t.fd.Read(expirations)
			if err == os.ErrClosed {
				break
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

func (t *Ticker) Reset(d time.Duration) bool {
	return t.ResetAt(-1, d)
}
func (t *Ticker) ResetAt(tt MonoTime, d time.Duration) bool

func NewTicker(d time.Duration) *Ticker {
	return NewTickerAt(-1, d)
}

func NewTickerAt(t MonoTime, d time.Duration) *Ticker
