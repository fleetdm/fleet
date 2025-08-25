package main

import (
	"flag"
	"fmt"
)

type Args struct {
	From     string
	To       string
	Debug    bool
	Help     bool
	Commands []string
}

func parseArgs() Args {
	var args Args

	// Override the default flag package's usage text.
	flag.Usage = func() {
		text, _ := usageText()
		fmt.Println(text)
	}

	// --from, -f
	flag.StringVar(&args.From, "from", "", "")
	flag.StringVar(&args.From, "f", "", "")

	// --to, -t
	flag.StringVar(&args.To, "to", "", "")
	flag.StringVar(&args.To, "t", "", "")

	// --debug
	flag.BoolVar(&args.Debug, "debug", false, "")

	// --help
	flag.BoolVar(&args.Help, "help", false, "")
	flag.BoolVar(&args.Help, "h", false, "")

	// Parse command-line inputs.
	flag.Parse()

	// Capture positional args.
	args.Commands = flag.Args()

	return args
}
