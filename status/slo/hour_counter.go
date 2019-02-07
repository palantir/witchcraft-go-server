package slo

import (
	"go.uber.org/atomic"
	"sync"
	"time"
)

// HourCounter tracks the number of calls to `Mark` over the course of the last hour.
type HourCounter struct {
	minutes []*MinuteCounter
}

// NewHourCounter instantiates and returns an HourCounter
func NewHourCounter() *HourCounter {
	h := HourCounter{
		minutes: make([]*MinuteCounter, 60),
	}
	for i := 0; i < 60; i++ {
		h.minutes[i] = &MinuteCounter{
			minuteIndex: i,
			counter: atomic.NewInt64(0),
		}
	}
	return &h
}

// Mark increments the counter given an instance in time.
func (h *HourCounter) Mark(t time.Time) {
	h.minutes[t.Minute()].Mark(t)
}

// Returns the exact number of times `Mark` has been called in the last hour
func (h *HourCounter) HourCount() int {
	var total int64
	for _, m := range h.minutes {
		total += m.Value()
	}
	return int(total)
}

// MinuteCounter tracks the number of calls to `Mark` over the course of one minute. It will automatically
// reset when `Mark` is called with a time.Time that is an hour or more in the future.
type MinuteCounter struct {
	validUntil time.Time
	validLock sync.Mutex

	minuteIndex int
	counter *atomic.Int64
}

// Mark increments the counter for number of times called in the minute configured. If the time provided is after
// the set expiry time, the counter is first reset before being incremented
func (m *MinuteCounter) Mark(t time.Time) {
	if t.Minute() != m.minuteIndex {
		return
	}
	m.resetIfNecessary()
	m.counter.Inc()
}

// Value returns the exact number of times `Mark` has been called for the given minute
func (m *MinuteCounter) Value() int64 {
	return m.counter.Load()
}

func (m *MinuteCounter) resetIfNecessary() {
	if !time.Now().After(m.validUntil) {
		return
	}
	m.validLock.Lock()
	defer m.validLock.Unlock()
	if !time.Now().After(m.validUntil) {
		return
	}
	m.validUntil = nextExpiry()
	m.counter.Store(0)
}

func nextExpiry() time.Time {
	return time.Now().Add(time.Minute * 60).Truncate(time.Minute)
}