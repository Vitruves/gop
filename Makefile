# Simple Makefile for GOP (Go Project) tool

# Go parameters
BINARY_NAME=gop
MAIN_PACKAGE=./cmd/gop

.PHONY: all build clean test run

# Default target
all: build

# Build the application
build:
	go build -buildvcs=false -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build successful!"

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	@echo "Clean successful!"

# Run tests
test:
	go test ./...
	@echo "Tests completed!"

# Run analyzer tests only
test-analyzer:
	go test ./internal/analyzer/tests/...
	@echo "Analyzer tests completed!"

# Format the code
fmt:
	go fmt ./...
	@echo "Formatting completed!"

# Run the application
run: build
	./$(BINARY_NAME)

# Run analyzers
todo:
	./$(BINARY_NAME) todo --directory=. --verbose

todo-file:
	./$(BINARY_NAME) todo --directory=. --output=todos.md --verbose

coherence:
	./$(BINARY_NAME) coherence --directory=. --verbose

concat:
	./$(BINARY_NAME) concat --directory=. --verbose

concat-file:
	./$(BINARY_NAME) concat --directory=. --output=concatenated.txt --verbose

registry:
	./$(BINARY_NAME) registry --directory=. --verbose

registry-file:
	./$(BINARY_NAME) registry --directory=. --output=registry.md --verbose

# Additional analyzers
complexity:
	./$(BINARY_NAME) complexity --directory=. --verbose

complexity-file:
	./$(BINARY_NAME) complexity --directory=. --output=complexity.md --verbose

api-usage:
	./$(BINARY_NAME) api-usage --directory=. --verbose

api-usage-file:
	./$(BINARY_NAME) api-usage --directory=. --output=api_usage.md --verbose

include-graph:
	./$(BINARY_NAME) include-graph --directory=. --verbose

include-graph-file:
	./$(BINARY_NAME) include-graph --directory=. --output=include_graph.md --verbose

call-graph:
	./$(BINARY_NAME) call-graph --directory=. --verbose

call-graph-file:
	./$(BINARY_NAME) call-graph --directory=. --output=call_graph.md --verbose

memory-safety:
	./$(BINARY_NAME) memory-safety --directory=. --verbose

memory-safety-file:
	./$(BINARY_NAME) memory-safety --directory=. --output=memory_safety.md --verbose

undefined-behavior:
	./$(BINARY_NAME) undefined-behavior --directory=. --verbose

undefined-behavior-file:
	./$(BINARY_NAME) undefined-behavior --directory=. --output=undefined_behavior.md --verbose

# Help target
help:
	@echo "Simple GOP Makefile"
	@echo "=================="
	@echo "Available targets:"
	@echo "  all        - Build the application (default)"
	@echo "  build      - Build the application"
	@echo "  clean      - Remove build artifacts"
	@echo ""
	@echo "Analyzer commands (output to console):"
	@echo "  todo                - Find TODOs in code"
	@echo "  complexity          - Analyze code complexity"
	@echo "  api-usage           - Analyze API usage"
	@echo "  include-graph       - Generate include dependency graph"
	@echo "  call-graph          - Generate function call graph"
	@echo "  memory-safety       - Check for memory safety issues"
	@echo "  undefined-behavior  - Detect undefined behavior"
	@echo "  registry            - Create code registry"
	@echo ""
	@echo "Analyzer commands (output to file):"
	@echo "  todo-file           - Find TODOs and save to todos.md"
	@echo "  complexity-file     - Analyze code complexity and save to complexity.md"
	@echo "  api-usage-file      - Analyze API usage and save to api_usage.md"
	@echo "  include-graph-file  - Generate include graph and save to include_graph.md"
	@echo "  call-graph-file     - Generate call graph and save to call_graph.md"
	@echo "  memory-safety-file  - Check memory safety and save to memory_safety.md"
	@echo "  undefined-behavior-file - Detect undefined behavior and save to undefined_behavior.md"
	@echo "  registry-file       - Create code registry and save to registry.md"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format the code"
	@echo "  run        - Run the application"
	@echo "  todo       - Run the TODO analyzer"
	@echo "  coherence  - Run the Coherence analyzer"
	@echo "  concat     - Run the Concat analyzer"
	@echo "  registry   - Run the Registry analyzer"
	@echo "  help       - Show this help message"
