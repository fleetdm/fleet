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
	"strconv"
	"strings"
	"time"

	"github.com/facebookincubator/flog"
	nvd "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

func extractCVSSBaseScore(item *Vulnerability) float64 {
	return strToFloat(item.CvssBaseScore)
}

func extractCVSSTemporalScore(item *Vulnerability) float64 {
	return strToFloat(item.CvssTemporalScore)
}

func extractCVSSVectorString(item *Vulnerability) string {
	return strings.Trim(item.CvssBaseVector, "()")
}

func extractCPEs(item *Vulnerability) []string {
	return strings.Split(item.CPE, ",")
}

func convertTime(fireeyeTime int64) string {
	return time.Unix(fireeyeTime, 0).Format(nvd.TimeLayout)
}

func strToFloat(str string) float64 {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		flog.Errorln(err)
		f = float64(0)
	}
	return f
}
