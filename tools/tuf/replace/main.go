package main

import (
	"os"
	"regexp"
	"strings"
)

// This tool was created to prevent issues between GNU's sed and OSX's sed.

func main() {
	inputPath := os.Args[1]
	expression := os.Args[2]
	replace := os.Args[3]
	r := regexp.MustCompile(expression)
	stat, err := os.Stat(inputPath)
	if err != nil {
		panic(err)
	}
	input, err := os.ReadFile(inputPath)
	if err != nil {
		panic(err)
	}
	if strings.HasSuffix(replace, `\n`) {
		replace = strings.TrimSuffix(replace, `\n`) + "\n"
	}
	output := r.ReplaceAllString(string(input), replace)
	if err := os.WriteFile(inputPath, []byte(output), stat.Mode()); err != nil {
		panic(err)
	}
}
