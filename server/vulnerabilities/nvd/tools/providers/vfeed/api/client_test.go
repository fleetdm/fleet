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
	"reflect"
	"testing"
)

const testDir = "testdata"

func TestClient(t *testing.T) {
	client := NewClient(testDir)

	items, err := client.FetchAllVulnerabilities(0)
	if err != nil {
		t.Fatalf("Fetch vulnerabilities failed: %v", err)
	}

	var got []string

	for item := range items {
		got = append(got, item.Information.Descriptions[0].ID)
	}

	want := []string{"CVE-2000-0001", "CVE-2001-0001"}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %s, got %s", want, got)
	}
}
