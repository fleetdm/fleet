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
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/facebookincubator/flog"
	"golang.org/x/sync/errgroup"
)

func main() {
	var o options
	o.addFlags()
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "wfnconvert provides various manipulations to WFN attributes")
		fmt.Fprintln(os.Stderr, "usage: wfnconvert [options]")
		fmt.Fprintln(os.Stderr, "options:")
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()
	if err := o.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "wfnconvert: %v\n\n", err)
		flag.Usage()
	}
	if err := wfnconvert(os.Stdin, os.Stdout, &o); err != nil {
		flog.Fatalf("wfnconvert: %v\n", err)
	}
}

func wfnconvert(in io.Reader, out io.Writer, o *options) error {
	inCh := make(chan []string)
	outCh := make(chan []string)

	g, ctx := errgroup.WithContext(context.Background())
	var inFunc, procFunc, outFunc func() error
	if len(o.csvFields) == 0 {
		inFunc = readLines(ctx, in, inCh)
		procFunc = processLines(ctx, o, inCh, outCh)
		outFunc = writeLines(out, outCh)
	} else {
		inFunc = readCSV(ctx, in, o.csvComma, inCh)
		procFunc = processCSV(ctx, o, inCh, outCh)
		outFunc = writeCSV(out, o.csvComma, outCh)
	}

	g.Go(func() error {
		defer close(inCh)
		return inFunc()
	})
	g.Go(func() error {
		defer close(outCh)
		return procFunc()
	})
	g.Go(outFunc)

	return g.Wait()
}

func readLines(ctx context.Context, in io.Reader, ch chan<- []string) func() error {
	scanner := bufio.NewScanner(in)
	return func() error {
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- []string{scanner.Text()}:
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("input error: %v", err)
		}
		return nil
	}
}

func readCSV(ctx context.Context, in io.Reader, comma string, ch chan<- []string) func() error {
	r := csv.NewReader(in)
	r.Comma = rune(comma[0])
	return func() error {
		for {
			ss, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("input error: %v", err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- ss:
			}
		}
		return nil
	}
}

func writeLines(out io.Writer, ch <-chan []string) func() error {
	return func() error {
		for ss := range ch {
			if len(ss) > 0 {
				fmt.Fprintln(out, ss[0])
			}
		}
		return nil
	}
}

func writeCSV(out io.Writer, comma string, ch <-chan []string) func() error {
	w := csv.NewWriter(out)
	w.Comma = rune(comma[0])
	return func() error {
		for ss := range ch {
			if err := w.Write(ss); err != nil {
				return fmt.Errorf("output error: %v", err)
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return fmt.Errorf("output error: %v", err)
		}
		return nil
	}
}

func processLines(ctx context.Context, o *options, in <-chan []string, out chan<- []string) func() error {
	return func() error {
		var (
			ss   []string
			more bool
		)
		for {
			select {
			case ss, more = <-in:
				if !more {
					return nil
				}
			case <-ctx.Done():
				return ctx.Err()
			}
			if len(ss) == 0 {
				continue
			}
			s, err := process(ss[0], o)
			if err != nil {
				return fmt.Errorf("could not process %q: %v", ss[0], err)
			}
			ss[0] = s
			select {
			case out <- ss:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func processCSV(ctx context.Context, o *options, in <-chan []string, out chan<- []string) func() error {
	return func() error {
		var (
			ss   []string
			more bool
		)
		for {
			select {
			case ss, more = <-in:
				if !more {
					return nil
				}
			case <-ctx.Done():
				return ctx.Err()
			}
			for _, n := range o.csvFields {
				if n >= len(ss) {
					continue
				}
				s, err := process(ss[n], o)
				if err != nil {
					return fmt.Errorf("could not process %q: %v", ss[n], err)
				}
				ss[n] = s
			}
			select {
			case out <- ss:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
