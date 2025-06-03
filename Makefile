# Simple Makefile for GOP (Go Project) tool

# Go parameters
BINARY_NAME=gop
MAIN_PACKAGE=./cmd/gop
INSTALL_PATH=/usr/local/bin

.PHONY: all build install clean help

# Default target
all: build

# Build the application (includes go tidy)
build:
	go mod tidy
	go build -buildvcs=false -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build successful!"

# Install the application
install: build
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "Installed $(BINARY_NAME) to $(INSTALL_PATH)"

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	@echo "Clean successful!"

# Help target
help:
	@echo "Simple GOP Makefile"
	@echo "=================="
	@echo "Available targets:"
	@echo "  all      - Build the application (default)"
	@echo "  build    - Build the application (includes go mod tidy)"
	@echo "  install  - Install the application to $(INSTALL_PATH)"
	@echo "  clean    - Remove build artifacts"
	@echo "  help     - Show this help message"
