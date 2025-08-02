# GitHub Manage - ghapi Package Tests

This directory contains comprehensive tests for all functions in the `ghapi` package.

## Test Files

- `cli_test.go` - Tests for command line interface functions
- `issues_test.go` - Tests for GitHub issues management functions  
- `models_test.go` - Tests for data models and struct conversion functions
- `projects_test.go` - Tests for GitHub Projects API functions
- `views_test.go` - Tests for view management and display functions
- `workflows_test.go` - Tests for bulk workflow operations
- `suite_test.go` - Comprehensive test suite runner and benchmarks

## Running Tests

### Run All Tests
```bash
cd tools/github-manage
go test ./pkg/ghapi -v
```

### Run Specific Test Files
```bash
go test ./pkg/ghapi -run TestCLI -v
go test ./pkg/ghapi -run TestIssues -v
go test ./pkg/ghapi -run TestModels -v
go test ./pkg/ghapi -run TestProjects -v
go test ./pkg/ghapi -run TestViews -v
go test ./pkg/ghapi -run TestWorkflows -v
```

### Run Test Suite
```bash
go test ./pkg/ghapi -run TestSuite -v
```

### Run Benchmarks
```bash
go test ./pkg/ghapi -bench=. -v
```

### Run Tests with Coverage
```bash
go test ./pkg/ghapi -cover -v
go test ./pkg/ghapi -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Categories

### Unit Tests
- **CLI Functions**: Test command execution and output parsing
- **JSON Parsing**: Test parsing of GitHub API responses
- **Data Models**: Test struct marshaling/unmarshaling and conversions
- **Project Management**: Test project field lookups and caching
- **View Management**: Test view creation and filtering
- **Workflow Operations**: Test bulk operations (currently stubs)

### Integration Tests
Some tests are skipped in the unit test suite because they require:
- GitHub CLI (`gh`) to be installed and authenticated
- Access to actual GitHub repositories and projects
- Network connectivity

To run integration tests, you need to:
1. Install and authenticate GitHub CLI
2. Have access to the Fleet repository
3. Run tests with appropriate environment setup

### Mocking
For proper integration testing, consider implementing mocks for:
- `RunCommandAndReturnOutput` function
- GitHub CLI commands
- External API calls

## Test Coverage

The tests cover:
- ✅ Basic function signatures and return types
- ✅ JSON parsing and error handling
- ✅ Data structure validation
- ✅ Struct conversions and transformations
- ✅ Caching mechanisms
- ✅ Error conditions and edge cases
- ⚠️ GitHub CLI interactions (mocked/skipped)
- ⚠️ External API calls (mocked/skipped)

## Notes

- Some tests are skipped on Windows due to bash command dependencies
- GitHub CLI-dependent tests are skipped without proper setup
- Stub functions (like `BulkSprintKickoff`) only test basic structure
- Consider implementing dependency injection for better testability
