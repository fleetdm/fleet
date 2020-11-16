Building The Code
=================

## Building the Code

Clone this repository.

To setup a working local development environment, you must install the following minimum toolset:

* [Go](https://golang.org/dl/) (1.9 or greater)
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

## Generating the packaged JavaScript

To generate all necessary code (bundling JavaScript into Go, etc), run the following:

```
make generate
```

### Automatic rebuilding of the JavaScript bundle

Normally, `make generate` takes the JavaScript code, bundles it into a single bundle via Webpack, and inlines that bundle into a generated Go source file so that all of the frontend code can be statically compiled into the binary. When you build the code after running `make generate`, all of that JavaScript is included in the binary.

This makes deploying Fleet a dream, since you only have to worry about a single static binary. If you are working on frontend code, it is likely that you don't want to have to manually re-run `make generate` and `make build` every time you edit JavaScript and CSS in order to see your changes in the browser. To solve this problem, before you build the Fleet binary, run the following command instead of `make generate`:

```
make generate-dev
```

Instead of reading the JavaScript from a inlined static bundle compiled within the binary, `make generate-dev` will generate a Go source file which reads the frontend code from disk and run Webpack in "watch mode".

Note that when you run `make generate-dev`, Webpack will be watching the JavaScript files that were used to generate the bundle, so the process will be long lived. Depending on your personal workflow, you might want to run this in a background terminal window.

After you run `make generate-dev`, run `make build` to build the binary, launch the binary and you'll be able to refresh the browser whenever you edit and save frontend code.

## Compiling the Fleet binary

Use `go build` to build the application code. For your convenience, a make command is included which builds the code:

```
make build
```

It's not necessary to use Make to build the code, but using Make allows us to account for cross-platform differences more effectively than the `go build` tool when writing automated tooling. Use whichever you prefer.

Once you're successful in building the code, head to the [development infrastrucutre](../development/development-infrastructure.md) documentation to use the local development Docker Compose infrastructure to run Fleet locally.
