package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/vulsio/goval-dictionary/commands"
)

// Name ... Name
const Name string = "goval-dictionary"

func main() {
	if envArgs := os.Getenv("GOVAL_DICTIONARY_ARGS"); 0 < len(envArgs) {
		commands.RootCmd.SetArgs(strings.Fields(envArgs))
	}

	if err := commands.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
