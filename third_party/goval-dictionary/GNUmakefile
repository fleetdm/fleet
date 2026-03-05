.PHONY: \
	all \
	build \
	install \
	lint \
	golangci \
	vet \
	fmt \
	mlint \
	fmtcheck \
	pretest \
	test \
	unused \
	cov \
	clean

SRCS = $(shell git ls-files '*.go')
PKGS = $(shell go list ./...)
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
LDFLAGS := -X 'github.com/vulsio/goval-dictionary/config.Version=$(VERSION)' \
	-X 'github.com/vulsio/goval-dictionary/config.Revision=$(REVISION)'
GO := CGO_ENABLED=0 go

all: build test

build: main.go
	$(GO) build -a -ldflags "$(LDFLAGS)" -o goval-dictionary $<

install: main.go
	$(GO) install -ldflags "$(LDFLAGS)"

lint:
	go install github.com/mgechev/revive@latest
	revive -config ./.revive.toml -formatter plain $(PKGS)

golangci:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

vet:
	echo $(PKGS) | xargs env $(GO) vet || exit;

fmt:
	gofmt -s -w $(SRCS)

fmtcheck:
	$(foreach file,$(SRCS),gofmt -s -d $(file);)

pretest: lint vet fmtcheck

test: pretest
	$(GO) test -cover -v ./... || exit;

cov:
	@ go get -v github.com/axw/gocov/gocov
	@ go get golang.org/x/tools/cmd/cover
	gocov test | gocov report

clean:
	echo $(PKGS) | xargs go clean || exit;
	echo $(PKGS) | xargs go clean || exit;
