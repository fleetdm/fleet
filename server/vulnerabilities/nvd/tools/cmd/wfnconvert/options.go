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
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
)

type intFields []int

// Set is part of flag.Value interface
func (f *intFields) Set(s string) error {
	if f == nil {
		return errors.New("Set called on nil receiver")
	}
	for _, t := range strings.Split(s, ",") {
		i, err := strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("failed to parse fields from %q: %v", s, err)
		}
		if i <= 0 {
			return fmt.Errorf("illegal CSV index %d: %q", i, s)
		}
		*f = append(*f, i-1)
	}
	return nil
}

// String is part of flag.Value interface
func (f *intFields) String() string {
	if f == nil {
		return ""
	}
	var sb strings.Builder
	for i, n := range *f {
		if i != 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%d", n+1)
	}
	return sb.String()
}

type strFields map[string]bool

// Set is part of flag.Value interface
func (f *strFields) Set(s string) error {
	if *f == nil {
		*f = make(strFields)
	}
	for _, s := range strings.Split(s, ",") {
		(*f)[s] = true
	}
	return nil
}

// String is part of flag.Value interface
func (f *strFields) String() string {
	if f == nil {
		return ""
	}
	var sb strings.Builder
	i := 0
	for k := range *f {
		if i != 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(k)
		i++
	}
	return sb.String()
}

type options struct {
	outBinding       string
	attributes       strFields
	invertAttributes bool
	any2na, na2any   bool
	csvFields        intFields
	csvComma         string
}

func (o *options) addFlags() {
	flag.StringVar(&o.outBinding, "b", "fstr", "output bindings, one of\n"+
		"'uri'\te.g. cpe:/a:foo:bar:1.1)\n"+
		"'fstr'\te.g. cpe:2.3:a:foo:bar:1.1:*:*:*:*:*:*:*)\n"+
		"'str'\te.g. "+`wfn:[part="a",vendor="foo",product="bar", version="1\.1"])`+"\n")
	flag.Var(&o.attributes, "attr", "comma-separated list of WFN attributes to manipulate; CPE 2.3 defines following attributes:\n"+
		"part, vendor, product, version, update, edition, language, sw_edition, target_sw, target_hw, other\n"+
		"additionally a special value 'all' is accepted")
	flag.BoolVar(&o.invertAttributes, "v", false, "invert attributes: process attributes not matched by -v flag")
	flag.BoolVar(&o.any2na, "any2na", false, "assign logical value N/A to all attributes with logical value ANY")
	flag.BoolVar(&o.na2any, "na2any", false, "assign logical value ANY to all attributes with logical value N/A")
	flag.Var(&o.csvFields, "csv", "comma-separated list of fields to read CPEs from, starting at 1;\n"+
		"assumes input as CSV, fields not specified in the list are passed unchanged")
	flag.StringVar(&o.csvComma, "d", ",", "if csv flag is set, sets the CSV delimiter, otherwise this flag is ignored")
}

func (o *options) validate() error {
	if err := validateAttrNames(o.attributes); err != nil {
		return err
	}

	if o.any2na && o.na2any {
		return fmt.Errorf("-any2na and -na2any flags are mutually exclusive")
	}

	switch o.outBinding {
	case "uri", "fstr", "str":
		break
	default:
		return fmt.Errorf("-b: invalid value %q", o.outBinding)
	}

	return nil
}

func (o *options) shouldProcess(name string) bool {
	do := o.attributes["all"] || o.attributes[name]
	if o.invertAttributes {
		return !do
	}
	return do
}

func (o *options) processAttributes(attr *wfn.Attributes, f func(*string) error) error {
	if o.shouldProcess("part") {
		if err := f(&attr.Part); err != nil {
			return err
		}
	}
	if o.shouldProcess("vendor") {
		if err := f(&attr.Vendor); err != nil {
			return err
		}
	}
	if o.shouldProcess("product") {
		if err := f(&attr.Product); err != nil {
			return err
		}
	}
	if o.shouldProcess("version") {
		if err := f(&attr.Version); err != nil {
			return err
		}
	}
	if o.shouldProcess("update") {
		if err := f(&attr.Update); err != nil {
			return err
		}
	}
	if o.shouldProcess("edition") {
		if err := f(&attr.Edition); err != nil {
			return err
		}
	}
	if o.shouldProcess("sw_edition") {
		if err := f(&attr.SWEdition); err != nil {
			return err
		}
	}
	if o.shouldProcess("target_sw") {
		if err := f(&attr.TargetSW); err != nil {
			return err
		}
	}
	if o.shouldProcess("target_hw") {
		if err := f(&attr.TargetHW); err != nil {
			return err
		}
	}
	if o.shouldProcess("other") {
		if err := f(&attr.Other); err != nil {
			return err
		}
	}
	if o.shouldProcess("language") {
		if err := f(&attr.Language); err != nil {
			return err
		}
	}
	return nil
}

func validateAttrNames(names strFields) error {
	valid := map[string]bool{
		"part":       true,
		"vendor":     true,
		"product":    true,
		"version":    true,
		"update":     true,
		"edition":    true,
		"language":   true,
		"sw_edition": true,
		"target_sw":  true,
		"target_hw":  true,
		"other":      true,
	}
	for n := range names {
		if n != "all" && !valid[n] {
			return fmt.Errorf("bad attribute name: %q", n)
		}
	}
	return nil
}
