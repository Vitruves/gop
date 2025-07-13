# gop - Go utilities for AI-assisted coding

Simple CLI tools to help with AI-assisted development workflows.

## Installation

```bash
go install github.com/vitruves/gop@latest
```

## Commands

### `gop concatenate`

Combine code files for AI analysis.

```bash
# Concatenate Python files
gop concatenate -l python

# Include build files, remove tests
gop concatenate -l rust --remove-tests -o output.txt
```

Options:
- `-l, --language` - Target language (python, rust, go, c, cpp)
- `--remove-tests` - Remove test files and test code
- `--remove-comments` - Strip comments
- `--add-line-numbers` - Add line numbers
- `--add-headers` - Add file path headers
- `-o, --output` - Output file

### `gop function-registry`

List functions in your codebase.

```bash
# Generate function list
gop function-registry -l go -o functions.md

# Find unused functions
gop function-registry -l python --only-dead-code

# Export to CSV for spreadsheet analysis
gop function-registry -l go -o functions.csv
```

Options:
- `-o, --output` - Output file (.md, .txt, .yaml, .json, .csv)
- `--by-script` - Group by file
- `--add-relations` - Show function calls
- `--only-dead-code` - Show unused functions only
- `--only-header-files` - C/C++ headers only

### `gop placeholders`

Find TODO comments and temporary code.

```bash
gop placeholders
```

Finds: TODO, FIXME, stub, temporary, hardcoded values, debug prints.

### `gop stats`

Show codebase statistics.

```bash
gop stats -o report.txt
```

## Global Options

- `-i, --include` - Include specific files/directories
- `-e, --exclude` - Exclude patterns
- `-R, --recursive` - Process subdirectories
- `-j, --jobs` - Number of parallel workers
- `-v, --verbose` - Show progress

## Examples

```bash
# Prepare Python code for AI review
gop concatenate -l python -R --remove-tests --add-headers

# Find all TODOs in project
gop placeholders -R

# Get project overview
gop stats -R

# List all Go functions
gop function-registry -l go --by-script
```

## Language Support

- **Python** (.py) + requirements.txt, setup.py
- **Rust** (.rs) + Cargo.toml, Cargo.lock  
- **Go** (.go) + go.mod, go.sum
- **C** (.c, .h) + Makefile, CMakeLists.txt
- **C++** (.cpp, .hpp, etc.) + build files

## License

MIT