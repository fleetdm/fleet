# Vulnerability Performance Testing Tools

This directory contains tools for testing the performance of Fleet's vulnerability-related datastore methods.

## Tools

### Seeder (`seeder/volume_vuln_seeder.go`)

Seeds the database with test data for performance testing.

**Usage:**

```bash
go run seeder/volume_vuln_seeder.go [options]
```

**Options:**

- `-hosts=N` - Number of hosts to create (default: 100)
- `-teams=N` - Number of teams to create (default: 5)
- `-cves=N` - Total number of unique CVEs in the system (default: 500)
- `-software=N` - Total number of unique software packages (default: 500)
- `-help` - Show help information
- `-verbose` - Enable verbose timing output for each step

**Example:**

```bash
go run seeder/volume_vuln_seeder.go -hosts=1000 -teams=10 -cves=2000 -software=4000
```

### Performance Tester (`tester/performance_tester.go`)

Benchmarks any Fleet datastore method with statistical analysis.

**Usage:**

```bash
go run tester/performance_tester.go [options]
```

**Options:**

- `-funcs=NAME[,NAME2,...]` - Comma-separated list of test functions (default: "UpdateVulnerabilityHostCounts")
- `-iterations=N` - Number of iterations per test (default: 5)
- `-verbose` - Show timing for each iteration
- `-details` - Show detailed statistics including percentiles
- `-list` - List available test functions
- `-help` - Show help information

**Available Test Functions:**

- `UpdateVulnerabilityHostCounts` - Test vulnerability host count updates

### Adding New Test Functions

To add support for additional datastore methods, edit the `testFunctions` map in `tester/performance_tester.go`:

```go
var testFunctions = map[string]TestFunction{
    // Existing functions...

    // Add new function
    "CountHosts": func(ctx context.Context, ds *mysql.Datastore) error {
        _, err := ds.CountHosts(ctx, fleet.TeamFilter{User: &fleet.User{}}, fleet.HostListOptions{})
        return err
    },

    // Add function with parameters
    "ListHosts:100": func(ctx context.Context, ds *mysql.Datastore) error {
        _, err := ds.ListHosts(ctx, fleet.TeamFilter{User: &fleet.User{}}, fleet.HostListOptions{
            ListOptions: fleet.ListOptions{Page: 0, PerPage: 100},
        })
        return err
    },
}
```

Each function should:

1. Accept `context.Context` and `*mysql.Datastore` as parameters
2. Return only an `error`
3. Handle any return values from the datastore method (discard non-error returns)
4. Use meaningful parameter values for realistic testing

**Examples:**

```bash
# Test single function with details
go run tester/performance_tester.go -funcs=UpdateVulnerabilityHostCounts -iterations=10 -details

# Test different batch sizes
go run tester/performance_tester.go -funcs=UpdateVulnerabilityHostCounts:5,UpdateVulnerabilityHostCounts:20 -iterations=5

# Verbose output
go run tester/performance_tester.go -funcs=UpdateVulnerabilityHostCounts -verbose
```

## Performance Analysis

The tools provide comprehensive performance metrics:

- **Total time** - Sum of all successful iterations
- **Average time** - Mean execution time
- **Min/Max time** - Fastest and slowest iterations
- **Success rate** - Percentage of successful vs failed iterations
- **Percentiles** - P50, P90, P99 response times (with `-details`)

## Typical Workflow

1. **Seed test data:**

   ```bash
   go run seeder/volume_vuln_seeder.go -hosts=1000 -teams=10 -cves=2000 -software=4000
   ```

2. **Test baseline performance:**

   ```bash
   go run tester/performance_tester.go -funcs=UpdateVulnerabilityHostCounts -iterations=10 -details
   ```

3. **Make code changes to optimize**

4. **Test optimized performance:**

   ```bash
   go run tester/performance_tester.go -funcs=UpdateVulnerabilityHostCounts -iterations=10 -details
   ```

5. **Compare results**

## Notes

- The seeder is not idempotent - run `make db-reset` to reset the database before reseeding.
