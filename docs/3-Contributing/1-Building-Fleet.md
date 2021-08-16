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

Install the dependencies as described in the next sections, then go to [Clone and build](#clone-and-build)

#### macOS

Enable the macOS developer tools:

```
xcode-select --install
```

Install [Homebrew](https://brew.sh/) to manage dependencies, then:

```
brew install git go node yarn
```

#### Ubuntu

Install dependencies:

```
sudo apt-get install -y git golang make nodejs npm
sudo npm install -g yarn
```

#### Windows

To install dependecies, we recommend using [Chocolatey](https://chocolatey.org/install). Chocolatey must be run in Powershell as an Administrator. Assuming your setup does not include any of our requirements, please run:
```
choco install nodejs git golang docker make python2
npm install -g yarn
```

Note: all packages default to the latest versions. To specify a version, place `--version <version-number>` after each package. You may also install all packages manually from their websites if you prefer.

After the packages have installed, you must use **Git Bash** to continue with the [next section](#clone-and-build).

### Clone and build

```
git clone https://github.com/fleetdm/fleet.git
cd fleet
make deps
make generate
make
```

The binaries are now available in `./build/`.

### Details

To setup a working local development environment, you must install the following minimum toolset:

* [Go](https://golang.org/doc/install)
* [Node.js](https://nodejs.org/en/download/current/) and [Yarn](https://yarnpkg.com/en/docs/install)
* [GNU Make](https://www.gnu.org/software/make/) (probably already installed if you're on macOS/Linux)

Once you have those minimum requirements, check out this [Loom video](https://www.loom.com/share/e7439f058eb44c45af872abe8f8de4a1) that walks through starting up a local development environment for Fleet.

For a text-based walkthrough, continue through the following steps:

First, you will need to install Fleet's dependencies.

To do this, run the following from the root of the repository:

```
make deps
```

When pulling changes, it may be necessary to re-run `make deps` if a new Go or JavaScript dependency was added.

### Generating the packaged JavaScript

To generate all necessary code (bundling JavaScript into Go, etc), run the following:

```
make generate
```

#### Automatic rebuilding of the JavaScript bundle

Normally, `make generate` takes the JavaScript code, bundles it into a single bundle via Webpack, and inlines that bundle into a generated Go source file so that all of the frontend code can be statically compiled into the binary. When you build the code after running `make generate`, all of that JavaScript is included in the binary.

This makes deploying Fleet a dream, since you only have to worry about a single static binary. If you are working on frontend code, it is likely that you don't want to have to manually re-run `make generate` and `make build` every time you edit JavaScript and CSS in order to see your changes in the browser. To solve this problem, before you build the Fleet binary, run the following command instead of `make generate`:

```
make generate-dev
```

Instead of reading the JavaScript from a inlined static bundle compiled within the binary, `make generate-dev` will generate a Go source file which reads the frontend code from disk and run Webpack in "watch mode".

Note that when you run `make generate-dev`, Webpack will be watching the JavaScript files that were used to generate the bundle, so the process will be long lived. Depending on your personal workflow, you might want to run this in a background terminal window.

After you run `make generate-dev`, run `make build` to build the binary, launch the binary and you'll be able to refresh the browser whenever you edit and save frontend code.

### Compiling the Fleet binary

For convenience, a Makefile is included to build the code:

```
make
```

It's not necessary to use Make to build the code, but using Make allows us to account for cross-platform differences more effectively than the `go build` tool when writing automated tooling. Use whichever you prefer.

## Development infrastructure

The following assumes that  [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) (installed by default with Docker on macOS and Windows) are installed.


### Starting the local development environment

To set up a canonical development environment via Docker, run the following from the root of the repository:

```
docker-compose up
```

##### Stopping the local development environment

If you'd like to shut down the virtual infrastructure created by Docker, run the following from the root of the repository:

```
docker-compose down
```

##### Setting up the database tables

Once you `docker-compose up` and are running the databases, you can build the code and run the following command to create the database tables:

```
./build/fleet prepare db --dev
```

### Running Fleet using Docker development infrastructure

To start the Fleet server backed by the Docker development infrastructure, run the Fleet binary as follows:

```
./build/fleet serve --dev
```

The server is accessible by default at [https://localhost:8080](https://localhost:8080). Note that `--dev` requires the use of `make generate-dev` as the server will not use bundled assets in this mode (you may see an error mentioning a template not found when visiting the website otherwise).

By default, Fleet will try to connect to servers running on default ports on `localhost`. Depending on your browser's settings, you may have to click through a security warning.

If you're using the Google Chrome web browser, you have the option to always automatically bypass the security warning. Visit [chrome://flags/#allow-insecure-localhost](chrome://flags/#allow-insecure-localhost) and set the "Allow invalid certificates for resources loaded from localhost." flag to "Enabled."

> Note: in Chrome version 88 there is a bug where you must first enable [chrome://flags/#temporary-unexpire-flags-m87](chrome://flags/#temporary-unexpire-flags-m87). The [chrome://flags/#allow-insecure-localhost](chrome://flags/#allow-insecure-localhost) flag will then be visible again.


## Debugging with Delve debugger

The [Delve](https://github.com/go-delve/delve) Go debugger can be used for debugging the Fleet binary.

Use the following command in place of `make` and `./build/fleet serve --dev`:

```
dlv debug --build-flags '-tags=full' ./cmd/fleet -- serve --dev
```

It is important to pass the `-tags=full` build flag, otherwise the server will not have access to the asset files.
