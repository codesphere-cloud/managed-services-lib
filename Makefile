.PHONY: all test cover lint fmt clean mocks build setup generate-notice

# Variables
GO=go
GOLANGCI_LINT=go tool golangci-lint

all: mocks fmt lint test build

# Test
test:
	$(GO) test -count=1 -v ./...

cover:
	$(GO) test -count=1 -coverprofile=coverage.out -coverpkg=./... \
		$(shell $(GO) list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...)
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint
lint:
	$(GOLANGCI_LINT) run

# Format
fmt:
	gofmt -w -s .

# Clean
clean:
	rm -rf bin/ dist/ coverage.out coverage.html

# Generate mocks
mocks:
	go tool mockery

# Compile check (this is a library — no binary is produced)
build:
	$(GO) build ./...

# Setup git hooks
setup:
	git config core.hooksPath .githooks

# Regenerate the third-party license NOTICE (requires the go-licenses tool dependency:
# go get -tool github.com/google/go-licenses)
generate-notice:
	go tool go-licenses report --template .NOTICE.template --ignore github.com/codesphere-cloud/managed-services-lib ./... > NOTICE