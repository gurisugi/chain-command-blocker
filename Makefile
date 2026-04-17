PLATFORMS := darwin_arm64 darwin_amd64 linux_amd64 linux_arm64
BINARIES := $(addprefix bin/chain-command-blocker-,$(PLATFORMS))

.PHONY: all build test clean FORCE

all: build

build: $(BINARIES)

bin/chain-command-blocker-%: FORCE
	@echo "Building $@..."
	@GOOS=$(word 1,$(subst _, ,$*)) GOARCH=$(word 2,$(subst _, ,$*)) \
		CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o $@ ./cmd/chain-command-blocker

FORCE:

test:
	@go test ./...

clean:
	rm -f $(BINARIES)
