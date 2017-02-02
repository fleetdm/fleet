.PHONY: build node-sass kolide-builder push

GO_VERSION = 1.8

all: clean build

build: node-sass kolide-builder

node-sass:
	mkdir -p build
	cd alpine-node-sass && \
		docker build -t node-sass-builder . && \
		docker run -v $(shell pwd)/build:/build node-sass-builder && cd ..

kolide-builder:
	# copy node-sass artifacts into rootfs
	cp build/* $(GO_VERSION)/rootfs/
	cd $(GO_VERSION) && \
		docker build -t kolide/kolide-builder:$(GO_VERSION) .

push:
		docker push kolide/kolide-builder:$(GO_VERSION)

clean: 
	rm -rf build
