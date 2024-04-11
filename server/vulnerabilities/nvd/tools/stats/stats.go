// Copyright (c) Facebook, Inc. and its affiliates.
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

package stats

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/facebookincubator/flog"
)

// Stats encapsulates functionallity of incrementing counters and incrementing values
type Stats struct {
	OutputFile   string
	LogToStderr  bool
	counters     map[string]int64
	countersLock sync.RWMutex
	values       map[string]float64
	valuesLock   sync.RWMutex
}

// New creates new Stats object
func New() *Stats {
	s := Stats{}
	s.Clear() // this will also initialize the maps
	return &s
}

// AreLogged returns whether this stats object is getting logged
// can be used to determine whether to keep adding stuff to it or not
// if it's not being logged, one shouldn't even increment counters
func (s *Stats) AreLogged() bool {
	return s.LogToStderr || s.OutputFile != ""
}

// AddFlags adds configuration flags for a stats object
func (s *Stats) AddFlags() {
	flag.StringVar(&s.OutputFile, "output_stats", "", "output stats to this file")
	flag.BoolVar(&s.LogToStderr, "log_stats", false, "log stats to stderr")
}

// IncrementCounter increments the counter associated with the key by 1
func (s *Stats) IncrementCounter(key string) {
	s.IncrementCounterBy(key, 1)
}

// IncrementCounterBy increments the counter associated with the key by the given value
func (s *Stats) IncrementCounterBy(key string, value int64) {
	s.countersLock.Lock()
	s.counters[key] += value
	s.countersLock.Unlock()
}

// AddToValue adds to the value associated with the key
func (s *Stats) AddToValue(key string, value float64) {
	s.valuesLock.Lock()
	s.values[key] += value
	s.valuesLock.Unlock()
}

// TrackTime will track how much time was elapsed from start time, and add that
// value (in given unit) to the value associated with the key
// should be used like this
// func Something() {
//   defer stats.TrackTime(key, time.Now(), time.Second)
//   .. do some operation
// }
//
func (s *Stats) TrackTime(key string, start time.Time, unit time.Duration) {
	elapsed := time.Since(start)
	value := float64(elapsed / unit)
	s.AddToValue(key, value)
}

// GetCounter returns the count associated with the key
func (s *Stats) GetCounter(key string) int64 {
	s.countersLock.RLock()
	defer s.countersLock.RUnlock()
	return s.counters[key]
}

// GetValue returns the value associated with the key
func (s *Stats) GetValue(key string) float64 {
	s.valuesLock.RLock()
	defer s.valuesLock.RUnlock()
	return s.values[key]
}

// Clear will empty out the stats, all counters are set to 0, all values set to 0
func (s *Stats) Clear() {
	s.counters = make(map[string]int64)
	s.values = make(map[string]float64)
}

// Write will write all stats to stderr and/or a file. configured through stats.OutputFile and stats.sLogToStderr
func (s *Stats) Write() error {
	if s.LogToStderr {
		if err := s.write(os.Stderr); err != nil {
			return fmt.Errorf("failed to write stats to stderr: %v", err)
		}
	}
	if s.OutputFile != "" {
		f, err := os.OpenFile(s.OutputFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open stats file: %v", err)
		}
		defer f.Close()
		if err = s.write(f); err != nil {
			return fmt.Errorf("failed to write stats to file: %v", err)
		}
	}
	return nil
}

// WriteAndLogError is just a wrapper around Write which also logs the error to stderr if it occurs
func (s *Stats) WriteAndLogError() {
	if err := s.Write(); err != nil {
		flog.Errorf("failed to write stats: %v", err)
	}
}

// global stats

var (
	// global is the global stats object. It can be used when you only need one stats object between multiple modules in a program
	global = New()
)

// AreLogged returns whether global stats are getting logged
func AreLogged() bool {
	return global.AreLogged()
}

// AddFlags adds configuration flags for the global stats object
func AddFlags() {
	global.AddFlags()
}

// IncrementCounter increments the global counter associated with the key by 1
func IncrementCounter(key string) {
	global.IncrementCounter(key)
}

// IncrementCounterBy increments the global counter associated with the key by the given value
func IncrementCounterBy(key string, value int64) {
	global.IncrementCounterBy(key, value)
}

// AddToValue adds to the value associated with the global key
func AddToValue(key string, value float64) {
	global.AddToValue(key, value)
}

// TrackTime will track how much time was elapsed from start time, and add that
// value (in given unit) to the global value associated with the key
// should be used like this
// func Something() {
//   defer stats.TrackTime(key, time.Now(), time.Second)
//   .. do some operation
// }
//
func TrackTime(key string, start time.Time, unit time.Duration) {
	global.TrackTime(key, start, unit)
}

// GetCounter returns the global count associated with the key
func GetCounter(key string) int64 {
	return global.GetCounter(key)
}

// GetValue returns the global value associated with the key
func GetValue(key string) float64 {
	return global.GetValue(key)
}

// Clear will empty out the global stats, all counters are set to 0, all values set to 0
func Clear() {
	global.Clear()
}

// Write will write all global stats to stderr and/or a file. configured through stats.OutputFile and stats.sLogToStderr
func Write() error {
	return global.Write()
}

// WriteAndLogError is just a wrapper around Write which also logs the error to stderr if it occurs
func WriteAndLogError() {
	global.WriteAndLogError()
}

// internal

func (s *Stats) write(w io.Writer) (err error) {
	cw := csv.NewWriter(w)

	defer func() {
		cw.Flush()
		if err == nil {
			// don't override the existing error with possible flush error
			err = cw.Error()
		}
	}()

	for key, counter := range s.counters {
		record := []string{key, fmt.Sprintf("%d", counter)}
		if err = cw.Write(record); err != nil {
			return err
		}
	}

	for key, value := range s.values {
		record := []string{key, fmt.Sprintf("%.2f", value)}
		if err = cw.Write(record); err != nil {
			return err
		}
	}

	return nil
}
