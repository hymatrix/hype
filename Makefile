BINARY := hype
BUILD_DIR := build
CMD := ./cmd/hype
GO := go

.PHONY: build clean install release

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD)

install:
	$(GO) install $(CMD)

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf $(BUILD_DIR) dist
