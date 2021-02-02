# Building Fleet
- [Building the code](#building-the-code)
  - [Generating the packaged JavaScript](#generating-the-packaged-javascript)
  - [Compiling the Fleet binary](#compiling-the-Fleet-binary)
- [Development infrastructure](#development-infrastructure)
  - [Starting the local development environment](#starting-the-local-development-environment)
  - [Running Fleet using Docker development infrastructure](#running-fleet-using-docker-development-infrastructure)
- [Setting up a Linux Development Environment](#setting-up-a-linux-development-environment)

## Building the code

Clone this repository.

To setup a working local development environment, you must install the following minimum toolset:

* [Go](https://golang.org/doc/install) (1.9 or greater)
* [Node.js](https://nodejs.org/en/download/current/) and [Yarn](https://yarnpkg.com/en/docs/install)
* [GNU Make](https://www.gnu.org/software/make/) (probably already installed if you're on macOS/Linux)
* [Docker](https://www.docker.com/products/overview#/install_the_platform)

> #### New to the Go language?
> 
> After installing Go, your $GOPATH will probably need a little freshening up.  To take care of this automatically every time a new terminal is opened, add this to your shell startup script (`~/.profile`):
> ```bash
> # Allow go-bindata and other Go stuff to work properly (e.g. for Fleet/osquery)
> # More info: https://golang.org/doc/gopath_code.html#GOPATH
> export PATH=$PATH:$(go env GOPATH)/bin
> ```

Once you have those minimum requirements, you will need to install Fleet's dependencies. To do this, run the following from the root of the repository:

```
make deps
```

When pulling in new revisions to your working source tree, it may be necessary to re-run `make deps` if a new Go or JavaScript dependency was added.

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

Use `go build` to build the application code. For your convenience, a make command is included which builds the code:

```
make build
```

It's not necessary to use Make to build the code, but using Make allows us to account for cross-platform differences more effectively than the `go build` tool when writing automated tooling. Use whichever you prefer.

## Development infrastructure

### Starting the local development environment

To set up a canonical development environment via docker, run the following from the root of the repository:

```
docker-compose up
```

This requires that you have docker installed. At this point in time, automatic configuration tools are not included with this project.

##### Stopping the local development environment

If you'd like to shut down the virtual infrastructure created by docker, run the following from the root of the repository:

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

The server is accessible by default at [https://localhost:8080](https://localhost:8080).

By default, Fleet will try to connect to servers running on default ports on localhost. Depending on your browser's settings, you may have to click through a security warning.

If you're using the Google Chrome web browser, you have the option to always automatically bypass the security warning. Visit [chrome://flags/#allow-insecure-localhost](chrome://flags/#allow-insecure-localhost) and set the "Allow invalid certificates for resources loaded from localhost." flag to "Enabled."

> Note: in Chrome version 88 there is a bug where you must first enable [chrome://flags/#temporary-unexpire-flags-m87](chrome://flags/#temporary-unexpire-flags-m87). The [chrome://flags/#allow-insecure-localhost](chrome://flags/#allow-insecure-localhost) flag will then be visible again.

If you're using Docker via [Docker Toolbox](https://www.docker.com/products/docker-toolbox), you may have to modify the default values use the output of `docker-machine ip` instead of `localhost`. There is an example configuration file included in this repository to make this process easier for you.  Use the `--config` flag of the Fleet binary to specify the path to your config. See `fleet --help` for more options.

## Setting up a Linux Development Environment

#### Install some dependencies

`sudo apt-get install xzip gyp libjs-underscore libuv1-dev dep11-tools deps-tools-cli`

#### Create a temp directory, download and place the `node` and `golang` bins 

```
mkdir tmp
cd tmp
```

#### install `node` and `yarn`

```
wget https://nodejs.org/dist/v9.4.0/node-v9.4.0-linux-x64.tar.xz
xz -d node-v9.4.0-linux-x64.tar.xz
tar -xf node-v9.4.0-linux-x64.tar
sudo cp -rf node-v9.4.0-linux-x64/bin /usr/local/
sudo cp -rf node-v9.4.0-linux-x64/include /usr/local
sudo cp -rf node-v9.4.0-linux-x64/lib /usr/local
sudo cp -rf node-v9.4.0-linux-x64/share /usr/local
npm install -g yarn
```

#### install `go`

```
wget https://dl.google.com/go/go1.9.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.9.3.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin:~/go/bin/
```

#### clean-up temp directory

```
cd ..
rm -rf tmp
```

#### Clone and build depenencies

```
git clone https://github.com/fleetdm/fleet.git
cd fleet
make deps
make generate
make build
sudo cp build/fleet /usr/bin/fleet
```
