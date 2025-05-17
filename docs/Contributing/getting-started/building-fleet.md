# Building Fleet
- [Building the code](#building-the-code)
  - [Quickstart](#quickstart)
  - [Clone and build](#clone-and-build)
  - [Details](#details)
- [Development infrastructure](#development-infrastructure)
  - [Starting the local development environment](#starting-the-local-development-environment)
  - [Running Fleet using Docker development infrastructure](#running-fleet-using-docker-development-infrastructure)
- [Debugging with Delve debugger](#debugging-with-delve-debugger)

## Building the code

### Quickstart

Install the dependencies as described in the following sections, then go to [Clone and build](#clone-and-build)

#### macOS

Enable the macOS developer tools:

```sh
xcode-select --install
```

Install [Homebrew](https://brew.sh/) to manage dependencies, then:

```sh
brew install git go node yarn
```

#### Ubuntu

Install dependencies:

```sh
sudo apt-get install -y git golang make nodejs npm
sudo npm install -g yarn
# Install nvm to manage node versions (apt very out of date) https://github.com/nvm-sh/nvm#install--update-script
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.5/install.sh | bash
# refresh your session before continuing
nvm install v20.18.1
```

#### Windows

To install dependencies, we recommend using [Chocolatey](https://chocolatey.org/install). Always run Chocolatey in Powershell as an Administrator. Assuming your setup does not include any of our requirements, please run:
```sh
choco install nodejs git golang docker make python2 mingw
npm install -g yarn
```

Note: all packages default to the latest versions. To specify a version, place `--version <version-number>` after each package. You may also install all packages manually from their websites if you prefer.

After installing the packages, you must use **Git Bash** to continue with the [next section](#clone-and-build).

If you plan to use [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) on your windows development environment, you can configure Docker to WSL integration by following the steps in [Microsoft's WSL Documentation](https://learn.microsoft.com/en-us/windows/wsl/tutorials/wsl-containers).

### Clone and build

```sh
git clone https://github.com/fleetdm/fleet.git
cd fleet
make deps
make generate
make
```

The binaries are now available in `./build/`.

### Details

To set up a working local development environment, you must install the following minimum toolset:

* [Go](https://golang.org/doc/install)
* [Node.js v20.18.1](https://nodejs.org/en/download/) and [Yarn](https://yarnpkg.com/en/docs/install)
* [GNU Make](https://www.gnu.org/software/make/) (probably already installed if you're on macOS/Linux)

Once you have those minimum requirements, check out this [Loom video](https://www.loom.com/share/e7439f058eb44c45af872abe8f8de4a1) that walks through starting up a local development environment for Fleet.

For a text-based walkthrough, continue through the following steps:

First, you will need to install Fleet's dependencies.

To do this, run the following from the root of the repository:

```sh
make deps
```

Note: If you are using python >= `3.12`, you may have to install `distutils` using pip.

```sh
pip install setuptools
```
or 
```sh
pip3 install setuptools
```

When pulling changes, it may be necessary to re-run `make deps` if a new Go or JavaScript dependency was added.

### Generating the packaged JavaScript

To generate all necessary code (bundling JavaScript into Go, etc.), run the following:

```sh
make generate
```

If you are using a Mac computer with Apple Silicon and have not installed Rosetta 2, you will need to do so before running `make generate`.

```sh
/usr/sbin/softwareupdate --install-rosetta --agree-to-license
```

#### Automatic rebuilding of the JavaScript bundle

Usually, `make generate` takes the JavaScript code, bundles it into a single bundle via Webpack, and inlines that bundle into a generated Go source file so that all of the frontend code can be statically compiled into the binary. When you build the code after running `make generate`, include all of that JavaScript in the binary.

This makes deploying Fleet a dream since you only have to worry about a single static binary. If you are working on frontend code, it is likely that you don't want to have to manually re-run `make generate` and `make build` every time you edit JavaScript and CSS in order to see your changes in the browser. Instead of running `make generate` to solve this problem, before you build the Fleet binary, run the following command:

```sh
make generate-dev
```

Instead of reading the JavaScript from an inline static bundle compiled within the binary, `make generate-dev` will generate a Go source file which reads the frontend code from disk and run Webpack in "watch mode."

Note that when you run `make generate-dev`, Webpack will be watching the JavaScript files that were used to generate the bundle so that the process will be long-lived. Depending on your personal workflow, you might want to run this in a background terminal window.

After you run `make generate-dev`, run `make build` to build the binary, launch the binary and you'll be able to refresh the browser whenever you edit and save frontend code.

### Compiling the Fleet binary

For convenience, Fleet includes a Makefile to build the code:

```sh
make
```

It's unnecessary to use Make to build the code, but using Make allows us to account for cross-platform differences more effectively than the `go build` tool when writing automated tooling. Use whichever you prefer.

## Development infrastructure

The following assumes that you already installed  [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) (installed by default with Docker on macOS and Windows).


### Starting the local development environment

To set up a canonical development environment via Docker, run the following from the root of the repository:

```sh
docker compose up
```

> Note: you can customize the DB Docker image via the environment variables FLEET_MYSQL_IMAGE and FLEET_MYSQL_PLATFORM. For example:
> - To run in macOS M1+, set FLEET_MYSQL_PLATFORM=linux/arm64/v8
> - To test with MariaDB, set FLEET_MYSQL_IMAGE to mariadb:10.6 or the like (note MariaDB is not officially supported).

### Stopping the local development environment

If you'd like to shut down the virtual infrastructure created by Docker, run the following from the root of the repository:

```sh
docker compose down
```

### Setting up the database tables

Once you `docker compose up` and are running the databases, you can build the code and run the following command to create the database tables:

```sh
./build/fleet prepare db --dev
```

### Running Fleet using Docker development infrastructure

To start the Fleet server backed by the Docker development infrastructure, run the Fleet binary as follows:

```sh
./build/fleet serve --dev
```

### Developing the Fleet UI

When the Fleet server is running, the Fleet UI is accessible by default at
[https://localhost:8080](https://localhost:8080).

> Note that `./build/fleet serve --dev` requires the use of `make generate-dev` because the server will not use bundled assets in this mode. (You may see an error mentioning a template not found when visiting the website otherwise.)

By default, Fleet will try to connect to servers running on default ports on `localhost`. Depending on your browser's settings, you may have to click through a security warning.

If you're using the Google Chrome web browser, you can always automatically bypass the security warning. Visit [chrome://flags/#allow-insecure-localhost](chrome://flags/#allow-insecure-localhost) and set the "Allow invalid certificates for resources loaded from localhost." flag to "Enabled."

> Note: in Chrome version 88, there is a bug where you must first enable
> [chrome://flags/#temporary-unexpire-flags-m87](chrome://flags/#temporary-unexpire-flags-m87). The
> [chrome://flags/#allow-insecure-localhost](chrome://flags/#allow-insecure-localhost) flag will
> then be visible again.

The Fleet UI is developed with [Typescript](https://www.typescriptlang.org/) using the [React library](https://reactjs.org/docs/getting-started.html) and [SCSS](https://sass-lang.com/) for styling.
The source code can be found in the [frontend](https://github.com/fleetdm/fleet/tree/main/frontend) directory.

## Debugging with Delve debugger

The [Delve](https://github.com/go-delve/delve) Go debugger can be used for debugging the Fleet binary.

Use the following command in place of `make` and `./build/fleet serve --dev`:

```sh
dlv debug --build-flags '-tags=full' ./cmd/fleet -- serve --dev
```

It is important to pass the `-tags=full` build flag; otherwise, the server will not have access to the asset files.

### Attaching a debugger to a running server

You can also run delve in headless mode, which allows you to attach your preferred debugger client and reuse the same session without having to restart the server:

```sh
dlv debug --build-flags '-tags=full' --headless \
  --api-version=2 --accept-multiclient --continue \
  --listen=127.0.0.1:61179 ./cmd/fleet -- serve --dev
```

- If you're using Visual Studio Code, there's a launch configuration in the repo.
- If you're using vim with `vimspector`, you can use the following config:

```json
{
  "configurations": {
    "Go: Attach to Fleet server": {
      "adapter": "multi-session",
      "variables": {
        "port": 61179,
        "host": "127.0.0.1"
      },
      "configuration": {
        "request": "attach",
        "mode": "remote"
      }
    }
  }
}
```

<meta name="pageOrderInSection" value="100">
<meta name="description" value="Learn about building Fleet from code, development infrastructure, and database migrations.">
