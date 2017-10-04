.PHONY: build fleet-builder push

GO_VERSION = 1.8

all: clean build

build: fleet-builder

fleet-builder:
	# copy node-sass artifacts into rootfs
	cd $(GO_VERSION) && \
		docker build -t kolide/fleet-builder:$(GO_VERSION) .

push:
		docker push kolide/fleet-builder:$(GO_VERSION)

clean:
	rm -rf build
