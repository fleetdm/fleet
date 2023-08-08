package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Open the file

	queriesPath := filepath.Join(".", "comandlines.txt")
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

		// If we encounter the separator "======", print the current section
		// and reset it for the next section.
		if strings.TrimSpace(line) == "===" {
			if len(section) > 0 {
				runQuery(section)
			}
			section = nil
		} else {
			// Add the line to the current section.
			section = append(section, line)
		}
	}

	// Print the last section (if there's any left after reading the file).
	if len(section) > 0 {
		runQuery(section)
	}

	// Check for any errors during scanning
	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file:", err)
	}
}

// Function to print a section (lines separated by "======")
func runQuery(section []string) {
	query := strings.Join(section, "\n")
	fmt.Println("Runningquery: " + query)

	cmd := exec.Command("orbit", "shell", query)
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

// // func cpuTimes() (float64, error) {
// // 	percent, err := cpu.Percent(0, false)
// // 	if err != nil {
// // 		return 0, err
// // 	}
// // 	return percent[0] / 100.0, nil
// // }

// 	// initialCPUTime, err := cpuTimes()
// DO SOMETHING
// 	// finalCPUTime, err := cpuTimes()

// 	// cpuUsage := finalCPUTime - initialCPUTime
// 	// fmt.Printf("CPU usage: %.2f%%\n", cpuUsage*100)
