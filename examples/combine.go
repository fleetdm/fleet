package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func walk(f *os.File) func(path string, info os.FileInfo, err error) error {
	return func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := f.WriteString(string(content) + "---\n"); err != nil {
			log.Fatal(err)
		}
		return nil
	}
}

func main() {
	inputDir := "./examples/config-intent-files"
	outputFile := "./examples/config-intent.yml"

	if err := os.Truncate(outputFile, 0); err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := filepath.Walk(inputDir, walk(f)); err != nil {
		log.Fatal(err)
	}
	f.Close()

	content, err := ioutil.ReadFile(outputFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile(outputFile, []byte(strings.Trim(string(content), "---\n")+"\n"), 0644); err != nil {
		log.Fatal(err)
	}
}
