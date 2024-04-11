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

package rpm

import (
	"testing"

	"github.com/facebookincubator/nvdtools/wfn"
)

func TestToWFN(t *testing.T) {
	cases := []struct {
		pkgName string
		cpe     string
		fail    bool
	}{
		{"", "", true},
		{"-1.0.1.x86_64", "", true},
		{"name-1.0-1.noarch.rpm", "cpe:2.3:a:*:name:1.0:1:*:*:*:*:*:*", false},
		{"NaMe-1.0-1.i386.rpm", "cpe:2.3:a:*:name:1.0:1:*:*:*:*:i386:*", false},
		{"NaMe-1.0-1.src.rpm", "cpe:2.3:a:*:name:1.0:1:*:*:*:*:*:*", false},
	}
	for _, c := range cases {
		var attr wfn.Attributes
		err := ToWFN(&attr, c.pkgName)
		if err != nil {
			if !c.fail {
				t.Errorf("%q: unexpected failure: %v", c.pkgName, err)
			}
			continue
		}
		if c.fail {
			t.Errorf("%q: unexpected success", c.pkgName)
			continue
		}
		if s := attr.BindToFmtString(); s != c.cpe {
			t.Errorf("%q: expected %q got %q", c.pkgName, c.cpe, s)
		}
	}
}

func BenchmarkToWFN(t *testing.B) {
	for i := 0; i < t.N; i++ {
		var attr wfn.Attributes
		ToWFN(&attr, "NaMe-1.0-1.i386.rpm")
	}
}
