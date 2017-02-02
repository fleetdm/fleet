Usage
```
Usage: builder.sh [args]
  -T,--tests            : Go run tests then exit
  -C,--ci               : Replicate full circle CI run
  -B,--build            : Build a release
```

### Caching pkg folder
If you're repeatedly testing the build on a development machine, it makes sense to mount the `$GOPATH/pkg` along with your source.
```
docker run --rm -it -v (pwd):/go/src/github.com/kolide/kolide-ose -v $GOPATH/pkg:/go/pkg kolide-builder -T
```
The first time the container runs, `go install` will compile all the dependencies under `$GOPATH/pkg/linux_amd64/...` making future test runs faster.

### Build a binary
Using the `-B,--build` flag will first run the CI build and then create a linux build in `./build/`.
This option is intended to be used followed by `docker build` to build a new release.

# Building the builder
use `make` to create a new container and then `make push` to push the builder to Docker Hub

The Makefile first compiles `node-sass` bindings to work on alpine linux and then builds the `kolide-builder` container with the compiled bindings.
Separating the two steps keeps the build container small, but increases the difficulty of building it.

