Building The Code
=================

## Installing build dependencies

To setup a working local development environment, you must install the following minimum toolset:

* [Go](https://golang.org/dl/) (1.8 or greater)
* [Node.js](https://nodejs.org/en/download/current/) (and npm)
* [GNU Make](https://www.gnu.org/software/make/)
* [Docker](https://www.docker.com/products/overview#/install_the_platform)


If you're using MacOS or Linux, Make should be installed by default. If you are using Windows, you will need to install it separately.

Once you have those minimum requirements, you will need to install Kolide's dependent libraries. To do this, run the following:

```
make deps
```

When pulling in new revisions to your working source tree, it may be necessary to re-run `make deps` if a new Go or JavaScript dependency was added.


```
make generate
```

## Generating the packaged JavaScript

To generate all necessary code (bundling JavaScript into Go, etc), run the following:

```
make generate
```

### Automatic rebuilding of the JavaScript bundle

Normally, `make generate` takes the JavaScript code, bundles it into a single bundle via Webpack, and inlines that bundle into a generated Go source file so that all of the frontend code can be statically compiled into the binary. When you build the code after running `make generate`, all of that JavaScript is included in the binary.

This makes deploying Kolide a dream, since you only have to worry about a single static binary. If you are working on frontend code, it is likely that you don't want to have to manually re-run `make generate` and `make build` every time you edit JavaScript and CSS in order to see your changes in the browser. To solve this problem, before you build the Kolide binary, run the following command instead of `make generate`:

```
make generate-dev
```

Instead of reading the JavaScript from a inlined static bundle compiled within the binary, `make generate-dev` will generate a Go source file which reads the frontend code from disk and run Webpack in "watch mode".

Note that when you run `make generate-dev`, Webpack will be watching the JavaScript files that were used to generate the bundle, so the process will be long lived. Depending on your personal workflow, you might want to run this in a background terminal window.

After you run `make generate-dev`, run `make build` to build the binary, launch the binary and you'll be able to refresh the browser whenever you edit and save frontend code.

## Compiling the Kolide binary

Use `go build` to build the application code. For your convenience, a make command is included which builds the code:

```
make build
```

It's not necessary to use Make to build the code, but using Make allows us to account for cross-platform differences more effectively than the `go build` tool when writing automated tooling. Use whichever you prefer.

## Managing Go Dependencies with Glide

[Glide](https://github.com/Masterminds/glide#glide-vendor-package-management-for-golang) is a package manager for third party Go libraries. See the ["How It Works"](https://github.com/Masterminds/glide#how-it-works) section in the Glide README for full details.

### Installing the correct versions of dependencies

To install the correct versions of third package libraries, use `glide install`. `glide install` will  use the `glide.lock` file to pull vendored packages from remote vcs.  `make deps` takes care of this step, as well as downloading the latest version of glide for you.

### Adding new dependencies

To add a new dependency, use [`glide get [package name]`](https://github.com/Masterminds/glide#glide-get-package-name)

### Updating dependencies

To update, use [`glide up`](https://github.com/Masterminds/glide#glide-update-aliased-to-up) which will use VCS and `glide.yaml` to figure out the correct updates.
