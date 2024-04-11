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

// b64schema converts a SQL schema file into base64 encoded strings as Go code.
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("use: %s [flags] input.sql output.go\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	pkg := flag.String("pkg", "schema", "set package name")
	flag.Parse()

	if len(flag.Args()) != 2 {
		flag.Usage()
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	o, err := os.Create(flag.Arg(1))
	if err != nil {
		panic(err)
	}
	defer o.Close()

	var b bytes.Buffer
	var stmt []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if b.Len() == 0 && line == "" {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
		if strings.HasSuffix(line, ";") {
			s := base64.StdEncoding.EncodeToString(b.Bytes())
			stmt = append(stmt, s)
			b.Reset()
		}
	}

	if len(stmt) == 0 {
		panic("empty stmt")
	}

	t := gotype(f.Name())

	decoderTemplate.Execute(o, struct {
		Pkg  string
		Pub  string
		File string
	}{
		*pkg, t, f.Name(),
	})

	fmt.Fprintf(o, "// b64%s is auto-generated from %s.\n", t, f.Name())
	fmt.Fprintf(o, "var b64%s = []string{", t)
	fmt.Fprintf(o, "%q", stmt[0])
	for i := 1; i < len(stmt); i++ {
		fmt.Fprintf(o, ", %q", stmt[i])
	}
	fmt.Fprintf(o, "}\n")
}

var decoderTemplate = template.Must(template.New("decoder").
	Parse(`package {{.Pkg}}

import (
	"context"
	"database/sql"
	"encoding/base64"
)

// Init{{.Pub}} is auto-generated. Executes each SQL statement from {{.File}}.
func Init{{.Pub}}(ctx context.Context, db *sql.DB) error {
	for _, stmt := range {{.Pub}}() {
		_, err := db.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

// {{.Pub}} is auto-generated. Returns each SQL statement from {{.File}}.
func {{.Pub}}() []string {
	s := make([]string, len(b64{{.Pub}}))
	for i := 0; i < len(s); i++ {
		v, _ := base64.StdEncoding.DecodeString(b64{{.Pub}}[i])
		s[i] = string(v)
	}
	return s
}

`))

var gotyper = regexp.MustCompile("[0-9A-Za-z]+")

func gotype(name string) string {
	parts := gotyper.FindAllString(name, -1)

	name = ""
	for _, part := range parts {
		p := strings.ToUpper(part)
		if _, exists := commonInitialisms[p]; exists {
			name += p
		} else {
			name += strings.Title(part)
		}
	}

	return name
}
