BINARY  := exam02
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: all build test vet dist clean

all: test build

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/exam02

test:
	go test ./...

vet:
	go vet ./...

# Cross-compiles every supported platform. cgo stays off so the binaries are
# static and the macOS builds need no Mac to produce.
dist: test
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -ldflags '$(LDFLAGS)' -o dist/$(BINARY)_$${os}_$${arch} ./cmd/exam02 || exit 1; \
	done
	@cd dist && shasum -a 256 $(BINARY)_* > SHA256SUMS 2>/dev/null || sha256sum $(BINARY)_* > SHA256SUMS
	@ls -lh dist/

clean:
	rm -rf dist $(BINARY)
