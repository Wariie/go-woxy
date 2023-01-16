# Usage:
# make        # title & compile all binary
# make run    # run code
# make build  # compile all binary
# make clean  # clean go project

.PHONY = all clean build install run vendor test

PROJECTNAME=$(shell basename "$(PWD)")

VERSION?=0.0.1
BINARY_NAME=go-woxy-$(VERSION)
DOCKER_REGISTRY?=

GOFILES=$(wildcard *.go)
GOCMD=go
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet

SERVICE_PORT=2000

STDERR=bin/go-woxy-stderr.txt
EXPORT_RESULT?=false

all: help

run:  ## Run project
	@echo Run
	$(GOCMD) run .

install: 
	$(GOCMD) install all

clean:  ## Remove build related file
	@rm -rf bin/
	@-$(MAKE) -s go-clean

vendor: ## Copy of all packages needed to support builds and tests in the vendor directory
	$(GOCMD) mod vendor

test: 
ifeq ($(EXPORT_RESULT), true)
	GO111MODULE=off go get -u github.com/jstemmer/go-junit-report
	$(eval OUTPUT_OPTIONS = | tee /dev/tty | go-junit-report -set-exit-code > junit-report.xml)
endif
	$(GOTEST) -v -race ./... $(OUTPUT_OPTIONS)

build:  ## Build your project and put the output binary in bin/
	@echo "  >  Setup building environment..."
	@mkdir bin 2> /dev/null || true
	@-touch $(STDERR)
	@-rm $(STDERR)
	@-$(MAKE) -s go-compile 2> $(STDERR)
	@cat $(STDERR) | sed -e '1s/.*/\nError:\n/'  | sed 's/make\[.*/ /' | sed "/^/s/^/     /" 1>&2

watch: ## Run the code with cosmtrek/air to have automatic reload on changes
	$(eval PACKAGE_NAME=$(shell head -n 1 go.mod | cut -d ' ' -f2))
	@echo $(PACKAGE_NAME) 
	docker run -it --rm -w /go/src/$(PACKAGE_NAME) -v $(shell pwd):/go/src/$(PACKAGE_NAME) -p $(SERVICE_PORT):$(SERVICE_PORT) cosmtrek/air

go-compile: go-clean go-build

go-build:
	@echo "  >  Building binaries"
	@echo "    >  Building for darwin os ..."
	GOARCH=amd64 GOOS=darwin $(GOCMD) build -o bin/${BINARY_NAME}-darwin .
	@echo "    >  Build for linux os ..."
	GOARCH=amd64 GOOS=linux $(GOCMD) build -o bin/${BINARY_NAME}-linux .
	@echo "    >  Build for windows os ..."
	$(GOCMD) build -o bin/${BINARY_NAME}-windows.exe .
	
go-generate:
	@echo "  >  Generating dependency files..."
	$(GOCMD) generate $(generate)

go-install:
	$(GOCMD) install $(GOFILES)

go-clean:
	@echo "  >  Cleaning build cache"
	$(GOCMD) clean

docker-build: ## Use the dockerfile to build the container
	docker build --rm --tag $(BINARY_NAME) .

docker-release: ## Release the container with tag latest and version
	docker tag $(BINARY_NAME) $(DOCKER_REGISTRY)$(BINARY_NAME):latest
	docker tag $(BINARY_NAME) $(DOCKER_REGISTRY)$(BINARY_NAME):$(VERSION)
	# Push the docker images
	docker push $(DOCKER_REGISTRY)$(BINARY_NAME):latest
	docker push $(DOCKER_REGISTRY)$(BINARY_NAME):$(VERSION)

help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo 'make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    %-20s%s\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  %s\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
