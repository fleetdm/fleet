# Upgrading the Go version used to build Fleet 

## Updating Go locally

The Go documentation doesn't include explicit instructions on upgrading versions. Some consider it a best practice to [completely uninstall the current version](https://go.dev/doc/manage-install#uninstalling) before installing a new one. If you used a package manager like Homebrew to install Go, you can upgrade it using `brew upgrade`. You can also install multiple versions by following the [Manage Go Installations guide](https://go.dev/doc/manage-install) in the Go docs. This is a good way to keep your previous version around for A/B testing.

If you use the golangci-lint linter in your local development, you'll need to recompile it using the new Go version:

```
go install github.com/golangci/golangci-lint/cmd/golangci-lint@<your golangci-lint version>
```

See the "Go linter" section below about upgrading the Go linter version.

## Files to update

### Go.mod

Update the Go version in [the main go.mod file in Fleet](https://github.com/fleetdm/fleet/blob/main/go.mod) to the new version.

Also update the version in the go.mod files of compiled tools including:

  * [Bitlocker manager](https://github.com/fleetdm/fleet/blob/main/tools/mdm/windows/bitlocker/go.mod)
  * [DB snapshot](https://github.com/fleetdm/fleet/blob/main/tools/snapshot/go.mod)
  * [Fleet teams terraform provider](https://github.com/fleetdm/fleet/blob/main/tools/terraform/go.mod)

Do a search for go.mod files to find others that may need updating!

### Go Linter

The golangci-lint linter used in the [golangci-lint](https://github.com/fleetdm/fleet/actions/workflows/golangci-lint.yml) Github action needs to support the new version of Go.  The linter typically keeps up with Go versions so that the latest version of golangci-lint should support the latest version of Go, but it's worth checking the [changelog](https://github.com/golangci/golangci-lint/blob/main/CHANGELOG.md) to see if it's time for an upgrade. If so, get the commit SHA of the version you'd like to use by finding [its tag on Github](https://github.com/golangci/golangci-lint/tags), clicking on the commit link under the tag name, and copying the full SHA from the URL.  Update the current `go install github.com/golangci/golangci-lint/cmd/golangci-lint` line in the .yml file with the new SHA, and update the comment to indicate the new version. 

Update the instructions in the [Testing and local development doc](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Testing-and-local-development.md#test-suite) to reflect the new version of golangci-lint.

### Docker files

Find the `bullseye` and `alpine` variants of the Docker images for the new Go version on [Dockerhub](https://hub.docker.com/_/golang).  Then update any Dockerfiles to pull the new images, including:

  * [Dockerfile-desktop-linux](https://github.com/fleetdm/fleet/blob/main/Dockerfile-desktop-linux)
  * [loadtest.Dockerfile](https://github.com/fleetdm/fleet/blob/main/infrastructure/loadtesting/terraform/docker/loadtest.Dockerfile_)
  * [mdmproxy/Dockerfile](https://github.com/fleetdm/fleet/blob/main/tools/mdm/migration/mdmproxy/Dockerfile)

## Smoke-testing the new version locally

1. Build `fleet` and `fleetctl` using `make build`. 
2. Verify that `fleet serve` runs, the site is accessible, and basic API/db functionality works (try creating a new team, query and policy)
3. Verify that `fleetctl` works by using `fleetctl get config`
4. Run `make go-lint` locally to find and fix any new issues.
5. Create a draft pull request from your branch and verify that tests pass.

## Updating this guide

As the Fleet project evolves, new areas may need to be touched when upgrading Go versions. Please update this guide with any new files you find that need changing (and remove any files that are no longer relevant).