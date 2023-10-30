package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fleetdm/orbit/pkg/table/dataflatten"
	"github.com/kolide/kit/logutil"
	"github.com/peterbourgon/ff/v3"
)

func checkError(err error) {
	if err != nil {
		fmt.Printf("Got Error: %v\nStack:\n%+v\n", err, err)
		os.Exit(1)
	}
}

func main() {
	flagset := flag.NewFlagSet("plist", flag.ExitOnError)

	var (
		flPlist = flagset.String("plist", "", "Path to plist")
		flJson  = flagset.String("json", "", "Path to json file")
		flXml   = flagset.String("xml", "", "Path to xml file")
		flIni   = flagset.String("ini", "", "Path to ini file")
		flQuery = flagset.String("q", "", "query")

		flDebug = flagset.Bool("debug", false, "use a debug logger")
	)

	if err := ff.Parse(flagset, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
	); err != nil {
		checkError(fmt.Errorf("parsing flags: %w", err))
	}

	logger := logutil.NewCLILogger(*flDebug)

	opts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(logger),
		dataflatten.WithNestedPlist(),
		dataflatten.WithQuery(strings.Split(*flQuery, `/`)),
	}

	rows := []dataflatten.Row{}

	if *flPlist != "" {
		data, err := dataflatten.PlistFile(*flPlist, opts...)
		if err != nil {
			checkError(fmt.Errorf("flattening plist file: %w", err))
		}
		rows = append(rows, data...)
	}

	if *flJson != "" {
		data, err := dataflatten.JsonFile(*flJson, opts...)
		if err != nil {
			checkError(fmt.Errorf("flattening json file: %w", err))
		}
		rows = append(rows, data...)
	}

	if *flXml != "" {
		data, err := dataflatten.XmlFile(*flXml, opts...)
		if err != nil {
			checkError(fmt.Errorf("flattening xml file: %w", err))
		}
		rows = append(rows, data...)
	}

	if *flIni != "" {
		data, err := dataflatten.IniFile(*flIni, opts...)
		if err != nil {
			checkError(fmt.Errorf("flattening ini file: %w", err))
		}
		rows = append(rows, data...)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "path", "parent key", "key", "value")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "----", "----------", "---", "-----")

	for _, row := range rows {
		p, k := row.ParentKey("/")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", row.StringPath("/"), p, k, row.Value)
	}
	w.Flush()

	return
}
