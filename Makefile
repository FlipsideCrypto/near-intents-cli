BINARY=near-intents
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test fmt vet ci clean dev-link

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -v ./...

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

vet:
	go vet ./...

ci: fmt vet test build

clean:
	rm -f $(BINARY)

dev-link: build
	ln -sf $(PWD)/$(BINARY) /usr/local/bin/$(BINARY)
