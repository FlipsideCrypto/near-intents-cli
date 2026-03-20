BINARY=near-intents
PORTFOLIO_BINARY=portfolio
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build build-portfolio test fmt vet ci clean dev-link dev-link-portfolio

build:
	go build $(LDFLAGS) -o $(BINARY) .

build-portfolio:
	go build $(LDFLAGS) -o $(PORTFOLIO_BINARY) ./cmd/portfolio

test:
	go test -v ./...

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

vet:
	go vet ./...

ci: fmt vet test build build-portfolio

clean:
	rm -f $(BINARY) $(PORTFOLIO_BINARY)

dev-link: build
	ln -sf $(PWD)/$(BINARY) /usr/local/bin/$(BINARY)

dev-link-portfolio: build-portfolio
	ln -sf $(PWD)/$(PORTFOLIO_BINARY) /usr/local/bin/$(PORTFOLIO_BINARY)
