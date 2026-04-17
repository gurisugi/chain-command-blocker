PLATFORMS := darwin_arm64 darwin_amd64 linux_amd64 linux_arm64
BINARIES := $(addprefix bin/chain-command-blocker-,$(PLATFORMS))

.PHONY: all build test lint pinact pinact-verify clean FORCE

all: build

build: $(BINARIES)

bin/chain-command-blocker-%: FORCE
	@echo "Building $@..."
	@GOOS=$(word 1,$(subst _, ,$*)) GOARCH=$(word 2,$(subst _, ,$*)) \
		CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o $@ ./cmd/chain-command-blocker

FORCE:

test:
	@go test ./...

TOOLS_MODFILE := tools/go.mod

lint:
	@go tool -modfile=$(TOOLS_MODFILE) golangci-lint run ./...

pinact:
	@go tool -modfile=$(TOOLS_MODFILE) pinact run

pinact-verify:
	@go tool -modfile=$(TOOLS_MODFILE) pinact run --check --verify

clean:
	rm -f $(BINARIES)
