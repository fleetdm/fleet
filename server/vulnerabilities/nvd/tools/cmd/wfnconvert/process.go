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

package main

import (
	"fmt"

	"github.com/facebookincubator/nvdtools/wfn"
)

func process(in string, o *options) (string, error) {
	attr, err := wfn.Parse(in)
	if err != nil {
		return "", fmt.Errorf("bad CPE %q: %v", in, err)
	}
	if o.any2na {
		o.processAttributes(attr, replaceAttributeValue(wfn.Any, wfn.NA))
	}
	if o.na2any {
		o.processAttributes(attr, replaceAttributeValue(wfn.NA, wfn.Any))
	}
	var out string
	switch o.outBinding {
	case "uri":
		out = attr.BindToURI()
	case "fstr":
		out = attr.BindToFmtString()
	case "str":
		out = attr.String()
	default:
		panic("bad output binding") // input is validated, shouldn't reach here
	}
	return out, nil
}

func replaceAttributeValue(src, dst string) func(*string) error {
	return func(s *string) error {
		if *s == src {
			*s = dst
		}
		return nil
	}
}
