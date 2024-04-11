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

package schema

import (
	"time"

	"github.com/facebookincubator/flog"
	nvd "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

var snykLayouts = []string{
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.000000Z",
}

func snykTimeToNVD(s string) string {
	var t time.Time
	var err error

	for _, layout := range snykLayouts {
		t, err = time.Parse(layout, s)
		if err == nil {
			return t.Format(nvd.TimeLayout)
		}
	}

	flog.Errorf("cannot parse snyk time: %v", err)
	return s
}
