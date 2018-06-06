GOPATH := $(GOPATH)
BINARY := ipfix-rita-converter

TESTFLAGS := -p=1 -v

# go source files
SRC := $(shell find . -path ./vendor -prune -o -type f -name '*.go' -print)

$(BINARY): vendor
	go build -o $(BINARY)

.PHONY: install
install: $(BINARY)
	mv $(BINARY) $(GOPATH)/bin/$(BINARY)

.PHONY: test
test: vendor docker-check
	go test ./...

vendor: Gopkg.lock
	dep ensure --vendor-only

Gopkg.lock: $(SRC) Gopkg.toml
	dep ensure --no-vendor

.PHONY: docker-check
docker-check:
	@if ! docker ps > /dev/null; then echo "Ensure docker is installed and accessible from the current user context"; return 1; fi
