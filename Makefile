NODE_BIN      = $(shell npm bin)
PID_FILE      = build/kolide.pid
GO_FILES      = $(filter-out ./bindata.go, $(shell find . -type f -name "*.go"))
TEMPLATES     = $(wildcard frontend/templates/*)

ifeq ($(OS), Windows_NT)
OUTFILE       = kolide.exe
else
OUTFILE       = kolide
endif

.prefix:
ifeq ($(OS), Windows_NT)
	if not exist build mkdir build
else
	mkdir -p build
endif

all: build

generate: .prefix
	go-bindata -pkg=app -o=app/bindata.go frontend/templates/ build/
	$(NODE_BIN)/webpack --progress --colors --bail

.PHONY: build
build: generate .prefix
	go build -o $(OUTFILE)

deps:
	npm install
	go get -u github.com/olebedev/on
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/tools/godep

clean:
	mkdir -p build
	rm -rf build/*

test: build
	go vet . ./app ./config ./errors ./sessions
	go test -v . ./app ./config ./errors ./sessions

stop:
	kill `cat $(PID_FILE)` || true

serve: .prefix
	BABEL_ENV=dev node hot.proxy &
	$(NODE_BIN)/webpack --watch &
	on -m 2 $(GO_FILES) $(TEMPLATES) | xargs -n1 -I{} make restart || make stop

restart: stop
	@echo restarting the app...
	$(TARGET) serve & echo $$! > $(PID_FILE)