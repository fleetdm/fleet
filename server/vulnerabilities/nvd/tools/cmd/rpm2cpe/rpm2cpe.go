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
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/rpm"
)

var progname = path.Base(os.Args[0])

// custom type to be recognized by flag.Parse()
type fieldsToSkip map[int]struct{}

// remove elements from  fields slice as per config
// NB!: this modifies the underlying array of fields slice
func (fs fieldsToSkip) skipFields(fields []string) []string {
	j := 0
	for i := 0; i < len(fields); i++ {
		if _, ok := fs[i]; ok {
			continue
		}
		fields[j] = fields[i]
		j++
	}
	return fields[:j]
}

// part of flag.Value interface implementation
func (fs fieldsToSkip) String() string {
	fss := make([]string, 0, len(fs))
	for i := range fs {
		fss = append(fss, fmt.Sprintf("%d", i+1))
	}
	return strings.Join(fss, ",")
}

// part of flag.Value interface implementation
func (fs *fieldsToSkip) Set(val string) error {
	if *fs == nil {
		*fs = fieldsToSkip{}
	}
	for _, v := range strings.Split(val, ",") {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		if n < 1 {
			return fmt.Errorf("illegal field index %d", n)
		}
		(*fs)[n-1] = struct{}{}
	}
	return nil
}

type config struct {
	rpmField    int
	cpeField    int
	inFieldSep  string
	outFieldSep string
	skip        fieldsToSkip
	defaultNA   bool
}

func (c *config) addFlags() {
	flag.IntVar(&c.rpmField, "rpm", 0, "position of the field in DSV input that contains the RPM name (starts at 1)")
	flag.IntVar(&c.cpeField, "cpe", 0, "position of the field in the output to put generated CPE at (starts at 1)")
	flag.StringVar(&c.inFieldSep, "d", "\t", "input column delimiter")
	flag.StringVar(&c.outFieldSep, "o", "\t", "output column delimiter")
	flag.Var(&c.skip, "e", "optional comma-separated list of input fields that should be dropped from output (starts with 1) "+
		"rpm name is extracted before dropping fields, CPE is added after that")
	flag.BoolVar(&c.defaultNA, "na", false, "if set, unknown CPE attributes are set to N/A, otherwise to ANY")
}

func sayErr(status int, msg string, args ...interface{}) {
	var statusStr string
	if status != 0 {
		statusStr = "fatal"
	} else {
		statusStr = "error"
	}
	fmt.Fprintf(os.Stderr, "%s: %s: %s\n", progname, statusStr, fmt.Sprintf(msg, args...))
	if status != 0 {
		os.Exit(status)
	}
}

func init() {
	var indent bytes.Buffer
	for i := 0; i < len(progname); i++ {
		indent.WriteByte(' ')
	}
	flag.Usage = func() {
		usageStr := "%[1]s takes a delimiter-separated input with one of the fields containing RPM package name\n" +
			"%[2]s and produces delimiter-separated output consisting of the same fields plus CPE name\n" +
			"%[2]s parsed from RPM package name.\n" +
			"usage: %[1]s [flags]\n" +
			"flags:\n"
		fmt.Fprintf(os.Stderr, usageStr, progname, indent.String())
		flag.PrintDefaults()
		os.Exit(1)
	}
}

// NB!: modifies underlying array of fields slice
func processRecord(fields []string, cfg config) ([]string, error) {
	if cfg.rpmField > len(fields) {
		return nil, fmt.Errorf("not enough fields (%d)", len(fields))
	}
	var attr *wfn.Attributes
	if cfg.defaultNA {
		attr = wfn.NewAttributesWithNA()
	} else {
		attr = wfn.NewAttributesWithAny()
	}
	attr.Vendor = wfn.Any
	if err := rpm.ToWFN(attr, fields[cfg.rpmField-1]); err != nil {
		return nil, fmt.Errorf("couldn't parse RPM name from field %q: %v", fields[cfg.rpmField-1], err)
	}
	cpe := attr.BindToURI()
	fields = cfg.skip.skipFields(fields)
	if cfg.cpeField > len(fields) {
		// if cfg.cpeField > len(fields)+1 we ignore that silently and just add CPE as the last field
		return append(fields, cpe), nil
	}
	outFields := make([]string, 0, len(fields)+1)
	outFields = append(outFields, fields[:cfg.cpeField-1]...)
	outFields = append(outFields, cpe)
	outFields = append(outFields, fields[cfg.cpeField-1:]...)
	return outFields, nil
}

func rpmname2cpe(in io.Reader, out io.Writer, cfg config) {
	r := csv.NewReader(in)
	r.Comma = rune(cfg.inFieldSep[0])
	w := csv.NewWriter(out)
	w.Comma = rune(cfg.outFieldSep[0])
	for {
		inRec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			sayErr(-1, "read error: %v", err)
		}
		outRec, err := processRecord(inRec, cfg)
		if err != nil {
			sayErr(0, "couldn't process record %v: %v", outRec, err)
			continue
		}
		if err = w.Write(outRec); err != nil {
			sayErr(-1, "write error: %v", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		sayErr(-1, "write error: %v", err)
	}
}

func main() {
	var cfg config
	cfg.addFlags()
	flag.Parse()
	if cfg.rpmField == 0 || cfg.cpeField == 0 {
		flag.Usage()
	}
	rpmname2cpe(os.Stdin, os.Stdout, cfg)
}
