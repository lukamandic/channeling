.PHONY: build run clean test example

build:
	@echo "Building channeling..."
	@go build -o channeling

run: build
	@echo "Running channeling..."
	@./channeling .

example: build
	@echo "Analyzing example file..."
	@./channeling examples

clean:
	@echo "Cleaning..."
	@rm -f channeling
	@go clean

deps:
	@echo "Installing dependencies..."
	@go mod tidy

help:
	@echo "Available commands:"
	@echo "  make build    - Build the CLI tool"
	@echo "  make run      - Build and run the tool (analyzes current directory)"
	@echo "  make example  - Build and run the tool on the example file"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make test     - Run tests"
	@echo "  make deps     - Install dependencies"
	@echo "  make help     - Show this help message" 