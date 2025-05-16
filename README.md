# GOP - Go Optimizer for Programming

GOP is a command-line tool designed to analyze, optimize, and refactor C/C++ code. It provides various tools to help developers improve code quality, generate documentation, analyze performance, and more.

## Features

### Code Metrics Tool
Calculates various code metrics for C/C++ code, including:
- Lines of code
- Cyclomatic complexity
- Function/method count
- Comment ratios

```bash
gop metrics --directory /path/to/code --output-file metrics.md
```

### Documentation Generator
Extracts documentation from code comments and generates Markdown documentation.

```bash
gop docs --directory /path/to/code --output-file documentation.md
```

### Performance Profiler
Analyzes the performance of executables using platform-specific tools (e.g., Instruments on macOS, perf/valgrind on Linux).

```bash
gop profile --executable /path/to/executable --type cpu
```

### Refactoring Tool
Assists with code refactoring by replacing patterns across multiple files, supporting both literal and regex matching.

```bash
gop refactor --directory /path/to/code --pattern "oldFunction" --replacement "newFunction"
```

## Installation

```bash
go install github.com/vitruves/gop@latest
```

Or build from source:

```bash
git clone https://github.com/vitruves/gop.git
cd gop
go build -o gop ./cmd/gop
```

## Usage

```bash
gop [command] [flags]
```

Available commands:
- `metrics`: Calculate code metrics
- `docs`: Generate documentation
- `profile`: Profile executable performance
- `refactor`: Refactor code
- `registry`: Extract code elements into a registry
- `todo`: Find TODO comments in code
- `undefined`: Analyze undefined behavior
- `concat`: Concatenate source files
- `call-graph`: Generate function call graphs
- `complexity`: Analyze code complexity
- `duplicate`: Find duplicate code
- `include`: Analyze include dependencies
- `style`: Check coding style

Use `gop help [command]` for more information about a specific command.

## Examples

### Calculate Code Metrics

```bash
gop metrics --directory ./src --languages c,cpp --output-file metrics.md
```

### Generate Documentation

```bash
gop docs --directory ./include --languages cpp --output-file docs.md
```

### Profile an Executable

```bash
gop profile --executable ./bin/myapp --type time --duration 10s
```

### Refactor Code

```bash
gop refactor --directory ./src --pattern "oldAPI" --replacement "newAPI" --regex
```

## Test Files

The project includes comprehensive test files in `internal/analyzer/tests/test_files/` to verify the functionality of all tools:

- `metrics_complex.cpp`: Tests complex code structures with high cyclomatic complexity
- `docs_well_documented.h`: Tests documentation extraction from well-documented code
- `docs_edge_cases.cpp`: Tests edge cases for documentation generation
- `refactor_candidate.c`: Tests refactoring with various code patterns
- `profile_performance.c`: Tests performance profiling with various algorithms
- `empty_file.c`: Tests handling of empty files
- `only_comments.cpp`: Tests handling of files with only comments
- `unicode_and_unusual.cpp`: Tests handling of Unicode characters and unusual syntax
- `complex_nested.cpp`: Tests handling of extremely complex nested structures

## License

[MIT License](LICENSE)
