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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/andreyvit/diff"

	"github.com/facebookincubator/flog"
	nvd "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

const (
	fixtureDir       = "testdata"
	inputFilePattern = "vfeed-example-*.json"
)

func mapToExpectedName(name string) string {
	return strings.Replace(name, "example", "converted", 1)
}

func withFixtureDir(name string) string {
	return path.Join(fixtureDir, name)
}

// mashalFile can be used to generate the "vfeed-converted.json" fixture.
func marshalFile(item interface{}, name string) error {
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(name, data, 0644)
}

func unmarshalFile(item interface{}, name string) error {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, item)
}

func mustMarshal(item interface{}) string {
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("MarshalIndent failed: %v", err))
	}

	return string(data)
}

func checkSchema(t *testing.T, input, expected string) {
	t.Helper()

	item := &Item{}
	if err := unmarshalFile(item, input); err != nil {
		flog.Fatalf("Failed to unmarshal example file: %v", err)
	}

	want := &nvd.NVDCVEFeedJSON10DefCVEItem{}
	if err := unmarshalFile(want, expected); err != nil {
		t.Fatalf("Failed to unmarshal converted file: %v", err)
	}

	got, err := item.Convert()
	if err != nil {
		t.Fatalf("Couldn't convert item: %v", err)
	}

	// TODO: using DeepEqual is a bit brittle.
	if !reflect.DeepEqual(want, got) {
		t.Fatalf(
			"Results differ (want, got): %s\n",
			diff.LineDiff(mustMarshal(want), mustMarshal(got)),
		)
	}
}

func TestSchema(t *testing.T) {
	matches, err := filepath.Glob(withFixtureDir(inputFilePattern))
	if err != nil {
		t.Fatalf("Couldn't match test files: %v", err)
	}

	if len(matches) == 0 {
		t.Fatalf("Expected test files, got nothing")
	}

	for _, match := range matches {
		match := match

		t.Run(match, func(t *testing.T) {
			checkSchema(t, match, mapToExpectedName(match))
		})
	}
}
