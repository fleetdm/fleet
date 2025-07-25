# Upgrading the Go version used to build Fleet 

## Updating Go locally

The Go documentation doesn't include explicit instructions on upgrading versions. Some consider it a best practice to [completely uninstall the current version](https://go.dev/doc/manage-install#uninstalling) before installing a new one. If you used a package manager like Homebrew to install Go, you can upgrade it using `brew upgrade`. You can also install multiple versions by following the [Manage Go Installations guide](https://go.dev/doc/manage-install) in the Go docs. This is a good way to keep your previous version around for A/B testing.

If you use the golangci-lint linter in your local development, you'll need to recompile it using the new Go version:

```
go install github.com/golangci/golangci-lint/cmd/golangci-lint@<your golangci-lint version>
```

See the "Go linter" section below about upgrading the Go linter version.

## Upgrading go for all Fleet

1. Run `make update-go version={target_version}` (for example, `make update-go version=1.24.5`)
2. Manually update the index digest sha256 of the updated Dockerfiles with the updated value of the
   relevant OS/Arch of the official Docker image from
   https://hub.docker.com/_/golang/tags?name=<image_for_the_dockerfile> (e.g.,
   `https://hub.docker.com/_/golang/tags?name=1.24.5-bullseye` when upgrading to 1.24.5)

### Go Linter

The golangci-lint linter used in the [golangci-lint](https://github.com/fleetdm/fleet/actions/workflows/golangci-lint.yml) Github action needs to support the new version of Go.  The linter typically keeps up with Go versions so that the latest version of golangci-lint should support the latest version of Go, but it's worth checking the [changelog](https://github.com/golangci/golangci-lint/blob/main/CHANGELOG.md) to see if it's time for an upgrade. If so, get the commit SHA of the version you'd like to use by finding [its tag on Github](https://github.com/golangci/golangci-lint/tags), clicking on the commit link under the tag name, and copying the full SHA from the URL.  Update the current `go install github.com/golangci/golangci-lint/cmd/golangci-lint` line in the .yml file with the new SHA, and update the comment to indicate the new version. 

Update the instructions in the [Testing and local development doc](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md#test-suite) to reflect the new version of golangci-lint.

## Smoke-testing the new version locally

1. Build `fleet` and `fleetctl` using `make build`. 
2. Verify that `fleet serve` runs, the site is accessible, and basic API/db functionality works (try creating a new team, query and policy)
3. Verify that `fleetctl` works by using `fleetctl get config`
4. Run `make lint-go` locally to find and fix any new issues.
5. Create a draft pull request from your branch and verify that tests pass.

## Updating this guide

As the Fleet project evolves, new areas may need to be touched when upgrading Go versions. Please update this guide with any new files you find that need changing (and remove any files that are no longer relevant).