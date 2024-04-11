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

// AdvisoryListResult type
type AdvisoryListResult struct {
	Count   int                    `json:"count"`
	Next    string                 `json:"next"`
	Previus string                 `json:"previous"`
	Results []*AdvisoryListElement `json:"results"`
}

// AdvisoryListElement type
type AdvisoryListElement struct {
	ID                 int64  `json:"id"`
	AdvisoryIdentifier string `json:"advisory_identifier"`
	Released           string `json:"released"`
	Modified           string `json:"modified_date"`
	// rest we don't need, it's included in the detail
}
