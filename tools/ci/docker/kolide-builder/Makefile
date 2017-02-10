.PHONY: build kolide-builder push

GO_VERSION = 1.8

all: clean build

build: kolide-builder

kolide-builder:
	# copy node-sass artifacts into rootfs
	cd $(GO_VERSION) && \
		docker build -t kolide/kolide-builder:$(GO_VERSION)-yarn .

push:
		docker push kolide/kolide-builder:$(GO_VERSION)-yarn

clean:
	rm -rf build
