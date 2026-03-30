BINARY := hype
BUILD_DIR := build
CMD := ./cmd/hype
GO := go
NPM := npm
FRONTEND_DIR := ./frontend

.PHONY: build clean install release snapshot frontend-build

frontend-build:
	$(NPM) --prefix $(FRONTEND_DIR) ci
	$(NPM) --prefix $(FRONTEND_DIR) run build

build: frontend-build
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD)

install: frontend-build
	$(GO) install $(CMD)

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf $(BUILD_DIR) dist
