SHS_VERSION := $(shell cat bin/SHS_VERSION)
PLATFORMS := darwin_arm64 darwin_amd64 linux_amd64 linux_arm64
BINARIES := $(addprefix bin/shs-,$(PLATFORMS))

.PHONY: all clean download-shs

all: download-shs

download-shs: $(BINARIES)

bin/shs-%: bin/SHS_VERSION
	@echo "Downloading shs $(SHS_VERSION) for $*..."
	@gh release download $(SHS_VERSION) -R gurisugi/shs -p "shs_$*.tar.gz" -D /tmp --clobber
	@tar -xzf /tmp/shs_$*.tar.gz -C /tmp
	@mv /tmp/shs $@
	@chmod +x $@
	@rm -f /tmp/shs_$*.tar.gz

clean:
	rm -f $(BINARIES)
