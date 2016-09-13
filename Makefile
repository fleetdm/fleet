.PHONY: build

all: build

.prefix:
ifeq ($(OS), Windows_NT)
	if not exist build mkdir build
else
	mkdir -p build
endif

build: .prefix
ifeq ($(OS), Windows_NT)
	go build -o build/kolide.exe
else
	go build -o build/kolide
endif

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
	go-bindata -pkg=server -o=server/bindata.go frontend/templates/ assets/ assets/images/
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

docker:
	docker pull kolide/kolide-builder:1.7
	docker run --rm -it -v $(shell pwd):/go/src/github.com/kolide/kolide-ose -v ${GOPATH}/pkg:/go/pkg kolide/kolide-builder:1.7 -B
	docker-compose up
