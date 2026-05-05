package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	_ "github.com/go-sql-driver/mysql"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
)

var (
	// MySQL config
	mysqlAddr = "localhost:3306"
	mysqlUser = "fleet"
	mysqlPass = "insecure"
	mysqlDB   = "fleet"
)

// TestFunction represents a datastore method to test
type TestFunction func(context.Context, *mysql.Datastore) error

// All available test functions
var testFunctions = map[string]TestFunction{
	"UpdateVulnerabilityHostCounts": func(ctx context.Context, ds *mysql.Datastore) error {
		return ds.UpdateVulnerabilityHostCounts(ctx, 1)
	},
}

// PerformanceResult holds the results of a performance test
type PerformanceResult struct {
	TestFunction         string
	TotalTime            time.Duration
	AverageTime          time.Duration
	MinTime              time.Duration
	MaxTime              time.Duration
	SuccessfulIterations int
	FailedIterations     int
	Iterations           []time.Duration
}

func runPerformanceTest(ctx context.Context, ds *mysql.Datastore, testFuncName string, iterations int, verbose bool) *PerformanceResult {
	testFunc, exists := testFunctions[testFuncName]
	if !exists {
		fmt.Printf("Unknown test function: %s\n", testFuncName)
		fmt.Printf("Available functions: %v\n", getTestFunctionNames())
		return nil
	}

	fmt.Printf("Running %d iterations of %s...\n", iterations, testFuncName)

	result := &PerformanceResult{
		TestFunction: testFuncName,
		Iterations:   make([]time.Duration, 0, iterations),
	}

	for i := 0; i < iterations; i++ {
		start := time.Now()

		if err := testFunc(ctx, ds); err != nil {
			log.Printf("Iteration %d failed: %v", i+1, err)
			result.FailedIterations++
			continue
		}

		duration := time.Since(start)
		result.Iterations = append(result.Iterations, duration)
		result.TotalTime += duration
		result.SuccessfulIterations++

		if verbose {
			fmt.Printf("Iteration %d: %v\n", i+1, duration)
		} else {
			fmt.Printf(".")
		}
	}

	if !verbose {
		fmt.Printf("\n")
	}

	if result.SuccessfulIterations == 0 {
		fmt.Printf("All iterations failed!\n")
		return result
	}

	// Calculate statistics
	result.AverageTime = result.TotalTime / time.Duration(result.SuccessfulIterations)

	// Find min and max
	result.MinTime = result.Iterations[0]
	result.MaxTime = result.Iterations[0]
	for _, duration := range result.Iterations {
		if duration < result.MinTime {
			result.MinTime = duration
		}
		if duration > result.MaxTime {
			result.MaxTime = duration
		}
	}

	return result
}

func printResults(results []*PerformanceResult, showDetails bool) {
	if len(results) == 0 {
		fmt.Printf("No results to display\n")
		return
	}

	fmt.Print("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("PERFORMANCE TEST RESULTS\n")
	fmt.Print(strings.Repeat("=", 80) + "\n\n")

	for _, result := range results {
		if result == nil {
			continue
		}

		fmt.Printf("Function: %s\n", result.TestFunction)
		fmt.Printf("  Total time:    %v\n", result.TotalTime)
		fmt.Printf("  Average time:  %v\n", result.AverageTime)
		fmt.Printf("  Min time:      %v\n", result.MinTime)
		fmt.Printf("  Max time:      %v\n", result.MaxTime)
		fmt.Printf("  Success rate:  %d/%d (%.1f%%)\n",
			result.SuccessfulIterations,
			result.SuccessfulIterations+result.FailedIterations,
			float64(result.SuccessfulIterations)/float64(result.SuccessfulIterations+result.FailedIterations)*100)

		if showDetails && len(result.Iterations) > 0 {
			fmt.Printf("  All times:     ")
			for i, duration := range result.Iterations {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%v", duration)
			}
			fmt.Printf("\n")

			// Calculate percentiles
			sortedTimes := make([]time.Duration, len(result.Iterations))
			copy(sortedTimes, result.Iterations)
			sort.Slice(sortedTimes, func(i, j int) bool {
				return sortedTimes[i] < sortedTimes[j]
			})

			if len(sortedTimes) >= 2 {
				p50 := sortedTimes[len(sortedTimes)/2]
				p90 := sortedTimes[int(float64(len(sortedTimes))*0.9)]
				p99 := sortedTimes[int(float64(len(sortedTimes))*0.99)]
				fmt.Printf("  P50:           %v\n", p50)
				fmt.Printf("  P90:           %v\n", p90)
				fmt.Printf("  P99:           %v\n", p99)
			}
		}
		fmt.Printf("\n")
	}
}

func getTestFunctionNames() []string {
	var names []string
	for name := range testFunctions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func main() {
	var (
		testFuncs  = flag.String("funcs", "UpdateVulnerabilityHostCounts", "Comma-separated list of test functions to run")
		iterations = flag.Int("iterations", 5, "Number of iterations per test function")
		verbose    = flag.Bool("verbose", false, "Show timing for each iteration")
		details    = flag.Bool("details", false, "Show detailed statistics including percentiles")
		listFuncs  = flag.Bool("list", false, "List available test functions and exit")
		help       = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *listFuncs {
		fmt.Printf("Available test functions:\n")
		for _, name := range getTestFunctionNames() {
			fmt.Printf("  %s\n", name)
		}
		return
	}

	if *help {
		fmt.Printf("Fleet Datastore Performance Tester\n\n")
		fmt.Printf("This tool measures the performance of Fleet datastore methods.\n")
		fmt.Printf("It assumes test data has already been seeded using the data seeding tool.\n\n")
		fmt.Printf("Available test functions:\n")
		for _, name := range getTestFunctionNames() {
			fmt.Printf("  %s\n", name)
		}
		fmt.Printf("\nExamples:\n")
		fmt.Printf("  %s -funcs=UpdateVulnerabilityHostCounts -iterations=10\n", os.Args[0])
		fmt.Printf("  %s -funcs=UpdateVulnerabilityHostCounts -iterations=5 -details\n", os.Args[0])
		fmt.Printf("  %s -funcs=UpdateVulnerabilityHostCounts -verbose\n", os.Args[0])
		fmt.Printf("\n")
		flag.Usage()
		return
	}

	ctx := context.Background()

	// Connect to datastore
	ds, err := mysql.New(config.MysqlConfig{
		Protocol: "tcp",
		Address:  mysqlAddr,
		Username: mysqlUser,
		Password: mysqlPass,
		Database: mysqlDB,
	}, clock.C)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = ds.Close() }()

	// Parse test functions
	funcNames := strings.Split(*testFuncs, ",")
	var results []*PerformanceResult

	for _, funcName := range funcNames {
		funcName = strings.TrimSpace(funcName)
		if funcName == "" {
			continue
		}

		result := runPerformanceTest(ctx, ds, funcName, *iterations, *verbose)
		results = append(results, result)
	}

	printResults(results, *details)
}
