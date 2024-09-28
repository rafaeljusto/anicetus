VCS_REF    = $(shell git rev-parse --short HEAD)
VERSION    = v$(shell git describe --always --match "v*")
TAG        = rafaeljusto/anicetus-http:$(VERSION)
LATEST_TAG = rafaeljusto/anicetus-http:latest

.PHONY: deploy build

default: build

build:
	docker build .

deploy:
	docker buildx build \
	  --platform linux/amd64,linux/arm64 \
	  --build-arg BUILD_VCS_REF=$(shell git rev-parse --short HEAD) \
	  --build-arg BUILD_VERSION=$(VERSION) \
	  -t $(TAG) \
	  -t $(LATEST_TAG) \
	  --push \
	  --progress=plain \
	  .