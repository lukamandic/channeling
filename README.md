# Channeling

A CLI tool written in Go that analyzes Go code and detects channel usage patterns.

## Features

- Detects channel declarations
- Identifies channel send and receive operations
- Provides detailed information about channel usage locations
- Supports analyzing entire directories of Go code

## Installation

1. Make sure you have Go installed (version 1.21 or later)
2. Clone this repository
3. Run the following command to install dependencies:
   ```bash
   go mod tidy
   ```
4. Build the tool:
   ```bash
   go build
   ```

## Usage

Run the tool by providing a directory path to analyze:

```bash
./channeling /path/to/your/go/project
```

The tool will analyze all Go files in the specified directory and its subdirectories, and print information about:

- Channel declarations
- Channel types
- Locations where channels are used (send/receive operations)
- File and line numbers for each usage

## Example Output

```
Channel Analysis Results:
========================

Channel: channel_123
Type: chan string
Location: /path/to/file.go:42
Usage:
  - Send at /path/to/file.go:45
  - Receive at /path/to/file.go:50
------------------------
```

## Requirements

- Go 1.21 or later
- github.com/spf13/cobra for CLI functionality
