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

package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const (
	ninetyDays = int64(60 * 60 * 24 * 90) // 90 days in seconds, max for FireEye api
)

type timeRangeParameters struct {
	StartDate int64
	EndDate   int64
}

func (vp timeRangeParameters) String() string {
	return fmt.Sprintf("Start: (%s); End: (%s)",
		time.Unix(vp.StartDate, 0).Format(time.RFC1123),
		time.Unix(vp.EndDate, 0).Format(time.RFC1123),
	)
}

func newParametersSince(since int64) timeRangeParameters {
	return timeRangeParameters{
		StartDate: since,
		EndDate:   time.Now().Unix(),
	}
}

func (vp timeRangeParameters) validate() error {
	if vp.StartDate < 0 {
		return fmt.Errorf("start date (%d) can't be < 0 ", vp.StartDate)
	}
	if vp.EndDate < 0 {
		return fmt.Errorf("end date (%d) can't be < 0 ", vp.EndDate)
	}
	if vp.EndDate < vp.StartDate {
		return fmt.Errorf("end date can't be < start date: %s", vp)
	}
	return nil
}

func (vp timeRangeParameters) query() string {
	query := url.Values{}
	query.Set("startDate", strconv.FormatInt(vp.StartDate, 10))
	query.Set("endDate", strconv.FormatInt(vp.EndDate, 10))
	return query.Encode()
}

func (vp timeRangeParameters) batchBy(gap int64) []timeRangeParameters {
	/*
		  This will create an array of parameters. Each element will have endDate - startDate <= gap
		  Actually, all but last elements will have endDate - startDate = gap
		  Last one might have <gap or =gap
			They will have no overlaps: [i+1].StartDate = [i].EndDate + 1

			[1,11].batchBy(3) = [(1,4),(5,8),(9,11)]
	*/
	var params []timeRangeParameters
	add := func(s, e int64) {
		params = append(params, timeRangeParameters{
			StartDate: s,
			EndDate:   e,
		})
	}
	current := vp.StartDate
	for {
		currentEnd := current + gap
		if currentEnd >= vp.EndDate {
			break
		}
		add(current, currentEnd)
		current = currentEnd + 1
	}
	add(current, vp.EndDate)
	return params
}
