.PHONY: build

all: build

.prefix:
ifeq ($(OS), Windows_NT)
	if not exist build mkdir build
else
	mkdir -p build
endif

ifeq ($(OS), Windows_NT)
OUTPUT 			= build/kolide.exe
else
OUTPUT 			= build/kolide
endif

VERSION			= 0.0.0-development
BRANCH			= $(shell git rev-parse --abbrev-ref HEAD)
REVISION		= $(shell git rev-parse HEAD)
GOVERSION		= $(shell go version | awk '{print $$3}')
NOW				= $(shell date +"%Y%m%d-%T")
USER			= $(shell whoami)
DOCKER_IMAGE_NAME = kolide/kolide

ifndef CIRCLE_PR_NUMBER
DOCKER_IMAGE_TAG = dev-unset
else
DOCKER_IMAGE_TAG = dev-${CIRCLE_PR_NUMBER}
endif

build: .prefix
	go build -o ${OUTPUT} -ldflags "\
-X github.com/kolide/kolide-ose/version.version=${VERSION} \
-X github.com/kolide/kolide-ose/version.branch=${BRANCH} \
-X github.com/kolide/kolide-ose/version.revision=${REVISION} \
-X github.com/kolide/kolide-ose/version.buildDate=${NOW} \
-X github.com/kolide/kolide-ose/version.buildUser=${USER} \
-X github.com/kolide/kolide-ose/version.goVersion=${GOVERSION}"

lint-js:
	$(shell npm bin)/eslint . --ext .js,.jsx

lint-go:
	go vet $(shell glide nv)

lint: lint-go lint-js

test-go:
	go test -v -cover $(shell glide nv)

test-js:
	$(shell npm bin)/_mocha --compilers js:babel-core/register --recursive 'frontend/**/*.tests.js*' --require 'frontend/.test.setup.js'

test: lint test-go test-js

generate: .prefix
	go-bindata -pkg=server -o=server/bindata.go frontend/templates/ assets/...
	$(shell npm bin)/webpack --progress --colors --bail

generate-dev: .prefix
	go-bindata -debug -pkg=server -o=server/bindata.go frontend/templates/ assets/...
	$(shell npm bin)/webpack --progress --colors --bail --watch

deps:
	npm install
	go get -u github.com/Masterminds/glide
	go get -u github.com/jteeuwen/go-bindata/...
	glide install

distclean:
	rm -rf build/*
	rm -rf assets/bundle.js
	rm -rf vendor/*

docker-build-circle:
	@echo ">> building docker image"
	docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
	docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"

docker:
	docker pull kolide/kolide-builder:1.7
	docker run --rm -it -v $(shell pwd):/go/src/github.com/kolide/kolide-ose -v ${GOPATH}/pkg:/go/pkg kolide/kolide-builder:1.7 -B
	docker-compose up
