# archtest

Architecture testing package for enforcing package dependency rules.

## Usage

```go
func TestMyPackageDependencies(t *testing.T) {
    archtest.NewPackageTest(t, "github.com/example/mypackage/...").
        OnlyInclude(regexp.MustCompile(`^github\.com/example/`)).
        WithTests().
        ShouldNotDependOn("github.com/example/forbidden/...").
        IgnoreDeps("github.com/example/infra/...").
        Check()
}
```

## Methods

| Method                       | Description                                                              |
|------------------------------|--------------------------------------------------------------------------|
| `OnlyInclude(regex)`         | Filter to only check packages matching the regex (for performance)       |
| `ShouldNotDependOn(pkgs...)` | Specify forbidden dependencies                                           |
| `IgnoreRecursively(pkgs...)` | Skip packages entirely (including their transitive deps); try not to use |
| `IgnoreDeps(pkgs...)`        | Allow packages but still check their transitive deps                     |
| `WithTests()`                | Include test imports from root packages only                             |
| `WithTestsRecursively()`     | Include test imports from all packages                                   |
| `IgnoreXTests(pkgs...)`      | Ignore external test packages (`_test` suffix)                           |
| `Check()`                    | Run the dependency check                                                 |
