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
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

func main() {
	acm := &AttributeColumnMap{}
	acm.AddFlags(flag.CommandLine)

	idx := flag.Int("i", 0, "column index (after erases) to insert cpe, zero means last")
	idelim := flag.String("d", ",", "input column delimiter")
	odelim := flag.String("o", "", "output column delimiter, optional")
	erase := flag.String("e", "", "comma separated list of columns to erase, optional")
	unmap := flag.Bool("x", false, "erase all columns mapped from -cpe_{field}, optional")
	lower := flag.Bool("lower", false, "force cpe output to be lower case, optional")
	defaultNA := flag.Bool("na", false, "if set, unknown CPE attributes are set to N/A, otherwise to ANY")

	flag.Parse()

	switch {
	case len(*idelim) != 1:
		fmt.Fprintln(os.Stderr, "input delimiter must be a single character")
		os.Exit(1)
	case len(*odelim) > 1:
		fmt.Fprintln(os.Stderr, "output delimiter must be a single character")
		os.Exit(1)
	case len(*odelim) == 0:
		*odelim = *idelim
	}

	var err error
	eraseCols := make(IntSet)

	if len(*erase) > 0 {
		eraseCols, err = NewIntSetFromString(strings.Split(*erase, ",")...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid column index passed to -e: %v", err)
		}
	}

	if *unmap {
		eraseCols.Merge(NewIntSet(acm.Columns()...))
	}

	p := &Processor{
		InputComma:        rune((*idelim)[0]),
		OutputComma:       rune((*odelim)[0]),
		CPEToLower:        *lower,
		DefaultNA:         *defaultNA,
		CPEOutputColumn:   *idx,
		EraseInputColumns: eraseCols,
	}

	err = p.Process(acm, os.Stdin, os.Stdout)
	if err != nil {
		flog.Fatalln(err)
	}
}

// Processor is a CSV processor.
type Processor struct {
	InputComma        rune   // input comma character
	OutputComma       rune   // output comma character
	CPEToLower        bool   // whether the output cpe should be forced lower case
	DefaultNA         bool   // true -> default attribute value is NA, false -> ANY
	CPEOutputColumn   int    // index to add cpe column in the output, after erases
	EraseInputColumns IntSet // input columns to erase before output
}

// Process reads CSV from r and writes CSV + CPE to w.
func (p *Processor) Process(acm *AttributeColumnMap, r io.Reader, w io.Writer) error {
	reader := csv.NewReader(r)
	reader.Comma = p.InputComma

	writer := csv.NewWriter(w)
	writer.Comma = p.OutputComma

	defer writer.Flush()

	line := 0

	for {
		line++

		cols, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error parsing line %d: %v", line, err)
		}

		cpe, err := acm.CPE(cols, p.CPEToLower, p.DefaultNA)
		if err != nil {
			return fmt.Errorf("error parsing columns in line %d: %v", line, err)
		}

		cols = RemoveColumns(cols, p.EraseInputColumns)
		cols = InsertColumn(cols, cpe, p.CPEOutputColumn)

		writer.Write(cols)
	}

	return nil
}

// AttributeColumnMap maps CSV columns to WFN Attribute fields.
type AttributeColumnMap struct {
	Part      int
	Vendor    int
	Product   int
	Version   int
	Update    int
	Edition   int
	SWEdition int
	TargetSW  int
	TargetHW  int
	Other     int
	Language  int
}

// AddFlags adds configuration flags to the given FlagSet.
func (acm *AttributeColumnMap) AddFlags(fs *flag.FlagSet) {
	fs.IntVar(&acm.Part, "cpe_part", 0, "part cpe column index")
	fs.IntVar(&acm.Vendor, "cpe_vendor", 0, "vendor cpe column index")
	fs.IntVar(&acm.Product, "cpe_product", 0, "product cpe column index")
	fs.IntVar(&acm.Version, "cpe_version", 0, "version cpe column index")
	fs.IntVar(&acm.Update, "cpe_update", 0, "update cpe column index")
	fs.IntVar(&acm.Edition, "cpe_edition", 0, "edition cpe column index")
	fs.IntVar(&acm.SWEdition, "cpe_swedition", 0, "swedition cpe column index")
	fs.IntVar(&acm.TargetSW, "cpe_targetsw", 0, "targetsw cpe column index")
	fs.IntVar(&acm.TargetHW, "cpe_targethw", 0, "targethw cpe column index")
	fs.IntVar(&acm.Other, "cpe_other", 0, "other cpe column index")
	fs.IntVar(&acm.Language, "cpe_language", 0, "language cpe column index")
}

// CPE returns a CPE by mapping cols to the configured column indices.
func (acm *AttributeColumnMap) CPE(cols []string, lower, na bool) (string, error) {
	var err error
	var attr *wfn.Attributes
	if na {
		attr = wfn.NewAttributesWithNA()
	} else {
		attr = wfn.NewAttributesWithAny()
	}

	m := map[int]*string{
		acm.Part:      &attr.Part,
		acm.Vendor:    &attr.Vendor,
		acm.Product:   &attr.Product,
		acm.Version:   &attr.Version,
		acm.Update:    &attr.Update,
		acm.Edition:   &attr.Edition,
		acm.SWEdition: &attr.SWEdition,
		acm.TargetSW:  &attr.TargetSW,
		acm.TargetHW:  &attr.TargetHW,
		acm.Other:     &attr.Other,
		acm.Language:  &attr.Language,
	}

	delete(m, 0)

	for i, v := range m {
		j := i - 1

		if j >= len(cols) {
			continue
		}

		col := cols[j]

		if lower {
			col = strings.ToLower(col)
		}

		if i == acm.Version {
			for strings.HasSuffix(col, ".") {
				col = strings.TrimSuffix(col, ".")
			}
		}

		*v, err = wfn.WFNize(col)
		if err != nil {
			return "", err
		}
	}

	return attr.BindToURI(), nil
}

// Columns returns a list of columns configured in the map, sorted descending.
func (acm *AttributeColumnMap) Columns() []int {
	s := NewIntSet(
		acm.Part,
		acm.Vendor,
		acm.Product,
		acm.Version,
		acm.Update,
		acm.Edition,
		acm.SWEdition,
		acm.TargetSW,
		acm.TargetHW,
		acm.Other,
		acm.Language,
	).ReverseSortedSet()

	for i, v := range s {
		if v == 0 {
			return s[:i]
		}
	}

	return s
}

// IntSet is a set of integers.
type IntSet map[int]struct{}

// NewIntSet creates and initializes a new IntSet.
func NewIntSet(s ...int) IntSet {
	m := make(IntSet, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

// NewIntSetFromString creates and initializes a new IntSet from
// indices and ranges in s. Example: 1-3,7,9 expands to 1,2,3,7,9.
func NewIntSetFromString(s ...string) (IntSet, error) {
	var err error
	islice := make([]int, 0, len(s))

	for _, str := range s {
		var start, end int

		p := strings.SplitN(str, "-", 2)
		if len(p) == 2 && p[1] != "" {
			end, err = strconv.Atoi(p[1])
			if err != nil {
				return nil, fmt.Errorf("failed to parse range %q: %v", str, err)
			}
		}

		start, err = strconv.Atoi(p[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse int: %q: %v", str, err)
		}

		if end == 0 {
			islice = append(islice, start)
			continue
		}

		if end <= start {
			return nil, fmt.Errorf("range end <= start: %q (%d <= %d)", str, end, start)
		}

		for i := start; i <= end; i++ {
			islice = append(islice, i)
		}
	}

	return NewIntSet(islice...), nil
}

// Merge merges ms into is.
func (is IntSet) Merge(ms IntSet) {
	for k, v := range ms {
		is[k] = v
	}
}

// ReverseSortedSet returns a reverse sorted set.
func (is IntSet) ReverseSortedSet() []int {
	s := make([]int, 0, len(is))
	for v := range is {
		s = append(s, v)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(s)))
	return s
}

// InsertColumn inserts column c to s at position idx.
// Invalid idx causes c to be appended to s.
func InsertColumn(s []string, c string, idx int) []string {
	if idx <= 0 || idx-1 >= len(s) {
		return append(s, c)
	}

	i := idx - 1
	return append(s[:i], append([]string{c}, s[i:]...)...)
}

// RemoveColumns returns a copy of cols with columns idx removed.
func RemoveColumns(cols []string, idx IntSet) []string {
	if len(cols) == 0 || len(idx) == 0 {
		return cols
	}

	nc := make([]string, 0, len(cols))

	for i, col := range cols {
		if _, exists := idx[i+1]; exists {
			continue
		}
		nc = append(nc, col)
	}

	return nc
}
