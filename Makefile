BINARY_NAME=kos
INSTALL_DIR=/usr/local/bin

# Build the binary
build:
 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY_NAME) main.go

# Install to system PATH (requires sudo)
install: build
 @echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
 sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
 @echo "✓ Installed! You can now use '$(BINARY_NAME)' from anywhere."

# Uninstall
uninstall:
 sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
 @echo "✓ Uninstalled."

# Clean build artifacts
clean:
 rm -f $(BINARY_NAME)

.PHONY: build install uninstall clean
