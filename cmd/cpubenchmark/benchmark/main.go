package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/shirou/gopsutil/cpu"
)

func main() {
	// Run the external executable
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path/to/executable>")
		return
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Get the initial CPU time
	initialCPUTime, err := cpuTimes()
	if err != nil {
		fmt.Println("Error getting CPU time:", err)
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Wait for the process to finish
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for process:", err)
	}

	// Get the final CPU time
	finalCPUTime, err := cpuTimes()
	if err != nil {
		fmt.Println("Error getting CPU time:", err)
	}

	// Calculate and print CPU usage
	cpuUsage := finalCPUTime - initialCPUTime
	fmt.Printf("CPU usage: %.2f%%\n", cpuUsage*100)
}

func cpuTimes() (float64, error) {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}
	return percent[0] / 100.0, nil
}
