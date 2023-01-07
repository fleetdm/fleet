package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
)

var (
	sourceFilePath  = path.Join(os.Getenv("GOROOT"), "src", "encoding", "asn1", "asn1.go")
	patchedFilePath = path.Join(os.Getenv("GOROOT"), "src", "encoding", "asn1", "asn1-patched.go")
)

func main() {
	// Check for the GOROOT env varible. Should be set by Go automatically
	if os.Getenv("GOROOT") == "" {
		panic("Plese set your GOROOT path")
	}

	// Load The file and create a scanner
	file, err := os.Open(sourceFilePath)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(file)

	// Open Output File
	out, err2 := os.Create(patchedFilePath)
	if err2 != nil {
		panic(err2)
	}

	// Loop of each line of the file checking it
	for scanner.Scan() {
		out.Write(scanner.Bytes())
		out.Write([]byte("\n"))

		if scanner.Text() == "		b == '?' ||" {
			scanner.Scan()
			if scanner.Text() != "		b == '!' || // Windows MDM Certificate Parsing Patch" {
				out.Write([]byte("		b == '!' || // Windows MDM Certificate Parsing Patch\n"))
				out.Write([]byte("		b == 0   || // Windows MDM Certificate Parsing Patch\n"))
			}

			out.Write(scanner.Bytes())
			out.Write([]byte("\n"))
		}
	}

	// Close writters
	file.Close()
	out.Close()

	// Replace the main file with the patched one
	if err := os.Rename(patchedFilePath, sourceFilePath); err != nil {
		panic(err)
	}

	// Success
	fmt.Println("Patch Applied To Your Go Sources! Please be carefull with the certs you are loading as they could cause undesired outcomes in the future.")
}
