package clock

import "time"

// Clock is a small abstraction for obtaining the current time.
// Use this in your application code to make time testable.
type Clock interface {
	Now() time.Time
}

// RealClock returns the real current time.
type RealClock struct{}

// Now returns the current time in UTC.
func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

// FakeClock is a simple controllable clock for tests.
type FakeClock struct {
	now time.Time
}

// NewFake creates a FakeClock set to the given time (expected in UTC).
func NewFake(t time.Time) *FakeClock {
	return &FakeClock{now: t}
}

// Now returns the fake current time.
func (f *FakeClock) Now() time.Time {
	return f.now
}

// Set sets the fake clock to a specific time.
func (f *FakeClock) Set(t time.Time) {
	f.now = t
}

// Advance moves the fake clock forward by duration d.
func (f *FakeClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}
