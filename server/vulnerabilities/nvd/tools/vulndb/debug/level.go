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

// Package debug provides debugging utilities.
package debug

import (
	"strconv"

	"github.com/pkg/errors"
)

// poor man's flog verbosity

// LevelFlag represents the verbosity level.
type LevelFlag int8

// Level is used for configuring the package verbosity.
var Level LevelFlag

// V reports whether verbosity at the call site is at least the
// requested level based on the configuration of Level.
func V(level LevelFlag) bool {
	return level <= Level
}

// Get implements the flag.Value interface.
func (l *LevelFlag) Get() interface{} {
	return *l
}

// Set implements the flag.Value interface.
func (l *LevelFlag) Set(value string) error {
	v, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return errors.Wrapf(err, "cannot convert verbosity level %q to int8", value)
	}

	*l = LevelFlag(v)
	return nil
}

// String implements the flag.Value interface.
func (l *LevelFlag) String() string {
	return strconv.Itoa(int(*l))
}

// Type implements the pflag.Value interface.
func (l *LevelFlag) Type() string {
	return "level"
}
