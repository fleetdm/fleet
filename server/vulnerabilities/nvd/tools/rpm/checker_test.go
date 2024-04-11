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
	"reflect"
	"testing"

	"github.com/facebookincubator/nvdtools/wfn"
)

func TestCheck(t *testing.T) {
	foo := "foo-v1-rel.arch.rpm"
	bar := "bar-v1-rel.arch.rpm"
	distro := "cpe:/o:vendor:product:version"

	// only foo has been fixed
	chk := nameChecker("foo")

	if c, err := Check(chk, foo, distro, ""); err != nil {
		t.Fatal(err)
	} else if !c {
		t.Fatal("expecting for foo to be fixed, but check returned false")
	}

	if c, err := Check(chk, bar, distro, ""); err != nil {
		t.Fatal(err)
	} else if c {
		t.Fatal("expecting for bar not to be fixed, but check returned true")
	}
}

func TestFilterFixedPackages(t *testing.T) {
	pkgs := []string{"foo-v1-rel.arch.rpm", "bar-v1-rel.arch.rpm", "malformed-rpm"}
	distro := "cpe:/o:vendor:product:version"

	// only foo has been fixed
	chk := nameChecker("foo")
	filtered, err := FilterFixedPackages(chk, pkgs, distro, "")
	if err != nil {
		t.Fatal(err)
	}
	// expecting to filter foo out and leave only bar and malformed
	filteredWant := []string{"bar-v1-rel.arch.rpm", "malformed-rpm"}
	if !reflect.DeepEqual(filtered, filteredWant) {
		t.Fatalf("wrong filtering\n\thave: %v\n\twant: %v", filtered, filteredWant)
	}
}

func TestCheckAnyAndAll(t *testing.T) {
	for i := 0; i <= 100; i++ {
		chks := getCheckers(i)
		// only false when all checkers are false
		if c := CheckAny(chks...).Check(nil, nil, ""); c == (i == 0) {
			t.Fatalf("unexpected ANY result for %d checkers: %v", i, c)
		}
		// only true when all checkers are true
		if c := CheckAll(chks...).Check(nil, nil, ""); c != isAllOnes(i) {
			t.Fatalf("unexpected ALL result for %d checkers: %v", i, c)
		}
	}
}

// Check method returns true if package name is the one specified
type nameChecker string

func (c nameChecker) Check(pkg *Package, _ *wfn.Attributes, _ string) bool {
	return pkg.Name == string(c)
}

// Check method returns whatever is the value of it
type constChecker bool

func (c constChecker) Check(_ *Package, _ *wfn.Attributes, _ string) bool {
	return bool(c)
}

// returns true and false checkers
// if we represent a number in binary format, then
//	- the number of true checkers is the number of 1s
//	- the number of false checkers is the number of 0s
func getCheckers(n int) []Checker {
	trueChk := constChecker(true)
	falseChk := constChecker(false)

	var chks []Checker
	for ; n > 0; n >>= 1 {
		if n&1 == 1 {
			chks = append(chks, trueChk)
		} else {
			chks = append(chks, falseChk)
		}
	}
	return chks
}

// is the number 2^x - 1
func isAllOnes(n int) bool {
	if n == 0 {
		return false
	}
	for n&1 == 1 {
		n >>= 1
	}
	return n == 0
}
