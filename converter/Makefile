GOPATH := $(GOPATH)
BINARY := ipfix-rita-converter
PREFIX := /usr/local

ETC := /etc/ipfix-rita
CONFIG := /converter.yaml

TESTFLAGS := -p=1 -v

# go source files
SRC := $(shell find . -path ./vendor -prune -o -type f -name '*.go' -print)

# Default build target is the executable
$(BINARY): vendor $(SRC)
	go build -o $(BINARY)

# START INSTALL
.PHONY: install
install: $(BINARY) root-check

ifeq (,$(wildcard $(ETC)$(CONFIG)))
install: install-config
endif

install:
	mv $(BINARY) $(PREFIX)/bin/$(BINARY)

.PHONY: install-config
install-config:
	mkdir -p "$(ETC)"
	cp etc/config.yaml "$(ETC)$(CONFIG)"

.PHONY: uninstall
uninstall: root-check
	rm $(PREFIX)/bin/$(BINARY)
	rm -rf $(ETC)
# END INSTALL

# BEGIN TESTING
.PHONY: test
test: vendor docker-check
	go test ./...
# END TESTING

# BEGIN VENDORING
vendor: Gopkg.toml $(SRC)
	dep ensure -v

#Note: Vendor isn't split because root cannot run dep ensure --no-vendor
#dep ensure --no-vendor may not touch Gopkg.lock if it is up to date.
#This leads to dep ensure --no-vendor almost always being run.
# END VENDORING

.PHONY: docker-check
docker-check:
	@if ! docker ps > /dev/null; then echo "Ensure docker is installed and accessible from the current user context"; return 1; fi

.PHONY: root-check
root-check:
	@if [ "$(shell id -u)" -ne 0 ]; then echo "This command requires administrator priviledges to proceed"; return 1; fi
