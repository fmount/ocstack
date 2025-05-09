# Variables
APP_NAME := ocstack
PKG := ./...

# Build the binary
build:
	go build -o bin/$(APP_NAME) main.go

# Run the app
run:
	go run main.go

# Run tests
test:
	go test -v $(PKG)

# Clean build artifacts
clean:
	rm -rf bin

# Format code
fmt:
	go fmt $(PKG)

vet:
	go vet $(PKG)

# Lint (optional, requires golangci-lint or similar tool)
lint: fmt vet
	golangci-lint run

tidy:
	go mod tidy

.PHONY: build run test clean fmt lint
