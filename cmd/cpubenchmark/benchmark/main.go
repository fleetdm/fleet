package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	args := os.Args
	queriesPath := args[1]
	orbitPath := args[2]

	numArgs := len(args) - 1
	if numArgs != 2 {
		fmt.Printf("Expecting two arguments. \nPath of queries files, \nPath of Orbit")
	}

	// Open the queries file
	file, err := os.Open(queriesPath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create a new scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Create a variable to hold the current section
	var section []string

	// Loop through each line and process the sections
	for scanner.Scan() {
		line := scanner.Text()

		// If we encounter the separator "===", run the current query
		// and reset it for the next section.
		if strings.TrimSpace(line) == "===" {
			if len(section) > 0 {
				runQuery(orbitPath, section)
			}
			section = nil
		} else {
			// Add the line to the current section.
			section = append(section, line)
		}
	}

	// Print the last section (if there's any left after reading the file).
	if len(section) > 0 {
		runQuery(orbitPath, section)
	}

	// Check for any errors during scanning
	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file:", err)
	}
}

// Function to run a query
func runQuery(orbitPath string, queryLines []string) {
	query := strings.Join(queryLines, "\n")
	fmt.Println("Runningquery: " + query)

	cmd := exec.Command(orbitPath, "shell", query)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()

	err := cmd.Start()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Wait for the process to finish
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for process:", err)
	}

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	fmt.Printf("Elapsed time: %v\n", elapsedTime)
}
