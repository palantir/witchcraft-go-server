// Copyright (c) 2021 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	decaysPerHalfLife = 10
)

var (
	decayFactor = math.Pow(0.5, 1.0/decaysPerHalfLife)
)

type CourseExponentialDecayReservoir interface {
	Update(updates float64)
	Get() float64
}

var _ CourseExponentialDecayReservoir = (*reservoir)(nil)

type reservoir struct {
	lastDecay                int64
	nanoClock                func() int64
	decayIntervalNanoseconds int64
	mu                       sync.Mutex
	value                    float64
}

func NewCourseExponentialDecayReservoir(nanoClock func() int64, halfLife time.Duration) CourseExponentialDecayReservoir {
	return &reservoir{
		lastDecay:                nanoClock(),
		nanoClock:                nanoClock,
		decayIntervalNanoseconds: halfLife.Nanoseconds() / decaysPerHalfLife,
	}
}

func (r *reservoir) Update(updates float64) {
	r.decayIfNecessary()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value = r.value + updates
}

func (r *reservoir) Get() float64 {
	r.decayIfNecessary()
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.value
}

func (r *reservoir) decayIfNecessary() {
	now := r.nanoClock()
	lastDecaySnapshot := r.lastDecay
	nanosSinceLastDecay := now - lastDecaySnapshot
	decays := nanosSinceLastDecay / r.decayIntervalNanoseconds
	// If CAS fails another thread will execute decay instead
	if decays > 0 && atomic.CompareAndSwapInt64(&r.lastDecay, lastDecaySnapshot, lastDecaySnapshot+decays*r.decayIntervalNanoseconds) {
		r.decay(decays)
	}
}

func (r *reservoir) decay(decayIterations int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value = r.value * math.Pow(decayFactor, float64(decayIterations))
}
