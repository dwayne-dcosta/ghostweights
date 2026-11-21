BINARY_NAME=ghostweights
BIN_DIR=bin

.PHONY: all build test clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cli
	@echo "✅ Built $(BIN_DIR)/$(BINARY_NAME)"

test:
	go test -v ./tests/...

clean:
	rm -rf $(BIN_DIR) *.gw
	@echo "🧹 Cleaned build artifacts"
