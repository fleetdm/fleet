package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/kolide/kit/logutil"
	"github.com/kolide/kit/version"
)

func runVersion(args []string) error {
	version.PrintFull()
	return nil
}

func runNoop(args []string) error {
	fmt.Printf("%+v\n", args)
	return nil
}

type runFunc func([]string) error
type subcommandMap map[string]runFunc
type commandMap map[string]subcommandMap

func usage() {
	fmt.Fprintf(os.Stderr, "fleetctl controls an instance of the Kolide Fleet osquery fleet manager.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Find more information at https://kolide.com/fleet\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  Usage:\n")
	fmt.Fprintf(os.Stderr, "    fleetctl [command] [flags]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  Commands:\n")
	fmt.Fprintf(os.Stderr, "    fleetctl query    - run a query across your fleet\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "    fleetctl apply    - apply a set of osquery configurations\n")
	fmt.Fprintf(os.Stderr, "    fleetctl edit    - edit your complete configuration in an ephemeral editor\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "    fleetctl help     - get help on how to define an intent type\n")
	fmt.Fprintf(os.Stderr, "    fleetctl version  - print full version information\n")
	fmt.Fprintf(os.Stderr, "\n")
}

func main() {
	logger := level.NewFilter(log.NewJSONLogger(os.Stderr), level.AllowDebug())
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	var run func([]string) error
	switch strings.ToLower(os.Args[1]) {
	case "version":
		run = runVersion
	case "query":
		run = runNoop
	case "edit":
		run = runNoop
	case "new":
		run = runNoop
	case "apply":
		run = runNoop
	case "help":
		run = runNoop
	default:
		usage()
		os.Exit(1)
	}

	if err := run(os.Args[2:]); err != nil {
		logutil.Fatal(logger, "err", err)
	}
}
