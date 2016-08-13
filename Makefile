WEBPACK = $(shell npm bin)/webpack --config=tools/app/webpack.config.js

.prefix:
ifeq ($(OS), Windows_NT)
	if not exist build mkdir build
else
	mkdir -p build
endif

generate: .prefix
	go-bindata -pkg=app -o=app/bindata.go frontend/templates/ build/
	$(WEBPACK) --progress --colors --bail

deps:
	npm install
	go get -u github.com/Masterminds/glide
	go get -u github.com/jteeuwen/go-bindata/...
ifneq ($(OS), Windows_NT)
	go get -u github.com/olebedev/on
endif
	# install vendored deps
	glide install

distclean:
	mkdir -p build
	rm -rf build/*
	rm -rf vendor/*

ifneq ($(OS), Windows_NT)

PID_FILE = build/kolide.pid
GO_FILES = $(filter-out ./bindata.go, $(shell find . -type f -name "*.go"))
TEMPLATES = $(wildcard frontend/templates/*)

stop:
	kill `cat $(PID_FILE)` || true

watch: .prefix
	BABEL_ENV=dev node tools/app/hot.proxy &
	$(WEBPACK) --watch &
	on -m 2 $(GO_FILES) $(TEMPLATES) | xargs -n1 -I{} make restart || make stop

restart: stop
	@echo restarting the app...
	kolide serve & echo $$! > $(PID_FILE)

endif
