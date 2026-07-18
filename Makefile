BINARY  := gs
CMD     := ./cmd
PKG     := main
VERSION ?= dev
GOOS    ?= $(shell go env GOOS)
GOARCH  ?= $(shell go env GOARCH)

BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S)
LDFLAGS    := -s -w \
	-X '$(PKG).BUILD_TIME=$(BUILD_TIME)' \
	-X '$(PKG).VERSION=$(VERSION)'

.PHONY: build run clean fmt vet test tidy release-build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

# Cross-compiled, reproducible build used by CI release matrix.
# Output name carries the target platform, e.g. gs-linux-amd64.
release-build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "$(LDFLAGS)" \
		-o dist/$(BINARY)-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,) $(CMD)

run:
	go run -ldflags "$(LDFLAGS)" $(CMD)

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

test:
	go test -race -coverprofile=coverage.out ./...

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist
