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

test:
	go vet $(shell glide nv)
	go test -v -cover $(shell glide nv)

generate: .prefix
	go-bindata -pkg=server -o=server/bindata.go frontend/templates/ build/
	$(shell npm bin)/webpack --progress --colors --bail

generate-dev: .prefix
	go-bindata -debug -pkg=server -o=server/bindata.go frontend/templates/ build/
	$(shell npm bin)/webpack --progress --colors --bail

deps:
	npm install
	go get -u github.com/Masterminds/glide
	go get -u github.com/jteeuwen/go-bindata/...
	glide install

distclean:
	rm -rf build/*
	rm -rf vendor/*

docker:
	docker pull kolide/kolide-builder:1.7
	docker run --rm -it -v $(shell pwd):/go/src/github.com/kolide/kolide-ose -v ${GOPATH}/pkg:/go/pkg kolide/kolide-builder:1.7 -B
	docker-compose up
