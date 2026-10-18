# KOS - Zero-Dependency Version Control
# Build and install with: make install

BINARY_NAME=kos
INSTALL_DIR=/usr/local/bin

.DEFAULT_GOAL := build
.PHONY: build install uninstall clean

# Build the binary
build:
	GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o $(BINARY_NAME) main.go

# Install to system PATH (requires sudo)
install: build
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	@echo "✅ KOS installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Run 'kos' from anywhere!"

# Uninstall from system
uninstall:
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ KOS uninstalled"

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
