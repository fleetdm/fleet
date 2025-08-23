package main

import "flag"

type Args struct {
	From     string
	To       string
	Debug    bool
	Commands []string
}

func parseArgs() Args {
	var args Args

	// TODO: CLI usage text && help text on these flags.

	// --from, -f
	flag.StringVar(&args.From, "from", "", "")
	flag.StringVar(&args.From, "f", "", "")

	// --to, -t
	flag.StringVar(&args.To, "to", "", "")
	flag.StringVar(&args.To, "t", "", "")

	// --debug
	flag.BoolVar(&args.Debug, "debug", false, "")

	// Parse command-line inputs.
	flag.Parse()

	// Capture positional args.
	args.Commands = flag.Args()

	return args
}
