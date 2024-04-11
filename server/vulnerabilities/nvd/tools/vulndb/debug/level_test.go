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

package debug

import (
	"flag"
	"testing"
)

func TestLevel(t *testing.T) {
	var level LevelFlag
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(&level, "v", "set verbosity")

	err := fs.Parse([]string{"-v=1"})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := level.Get().(LevelFlag); !ok {
		t.Fatal("unexpected type")
	}

	switch {
	case level != 1:
		t.Fatal("unexpected level")
	case level.Type() != "level":
		t.Fatal("unexpected level type")
	}

	if V(1) {
		t.Fatal("unexpected global level")
	}
}
