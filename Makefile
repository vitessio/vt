.PHONY: all build test tidy clean pretty install install-tools lint install-hooks
.DEFAULT_GOAL := test_and_build

REQUIRED_GO_VERSION := 1.23
GOLANGCI_LINT_VERSION := v2.0.2

# Determine the Go binary directory
GOBIN_DIR := $(or $(GOBIN), $(shell go env GOBIN))
ifeq ($(GOBIN_DIR),)
	GOBIN_DIR := $(shell go env GOPATH)/bin
endif

COMMIT := $(shell git rev-parse --short HEAD)
DATE   := $(shell date +%Y-%m-%dT%H:%M:%S)

test_and_build: test build

# Version check
check_version:
	@GO_VERSION=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	MAJOR_VERSION=$$(echo $$GO_VERSION | cut -d. -f1); \
	MINOR_VERSION=$$(echo $$GO_VERSION | cut -d. -f2); \
	if [ "$$MAJOR_VERSION" -eq 1 ] && [ "$$MINOR_VERSION" -lt 23 ]; then \
		echo "Error: Go version $(REQUIRED_GO_VERSION) or higher is required. Current version is $$GO_VERSION"; \
		exit 1; \
	else \
		echo "Go version is acceptable: $$GO_VERSION"; \
	fi

default: check_version build

build:
	@echo "Building vt..."
	@go build -ldflags "\
	  -X github.com/vitessio/vt/go/cmd.CommitSha=$(COMMIT) \
	  -X github.com/vitessio/vt/go/cmd.BuildDate=$(DATE)" \
	  -o vt ./go/vt

test:
	go test -v -count=1 ./go/...

tidy:
	go mod tidy

clean:
	go clean -i ./...
	rm -f vt

# Pretty: formats the code using gofumpt and goimports-reviser
pretty: check-tools
	@echo "Running formatting tools..."
	@gofumpt -w . >/dev/null 2>&1
	@goimports-reviser -recursive -project-name $$(go list -m) -rm-unused -set-alias ./go >/dev/null 2>&1

# Tools installation command
install-tools:
	@echo "Installing gofumpt..."
	go install mvdan.cc/gofumpt@latest

	@echo "Installing goimports-reviser..."
	go install github.com/incu6us/goimports-reviser/v3@latest

	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $(GOBIN_DIR) $(GOLANGCI_LINT_VERSION)

	@echo "All tools installed successfully."

# Ensure tools are available
check-tools:
	@command -v gofumpt >/dev/null 2>&1 || { echo "gofumpt not found. Run 'make install-tools' to install it." >&2; exit 1; }
	@command -v goimports-reviser >/dev/null 2>&1 || { echo "goimports-reviser not found. Run 'make install-tools' to install it." >&2; exit 1; }
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci_lint not found. Run 'make install-tools' to install it." >&2; exit 1; }


# Lint: runs golangci-lint
lint: check-tools
	@echo "Running golangci-lint..."
	@golangci-lint run --config .golangci.yml ./go/...

install-hooks:
	@echo "Installing Git hooks..."
	@ln -sf ../../git-hooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed successfully."

install: build
	@install -m 0755 vt $(GOBIN_DIR)/vt
	@echo "vt installed successfully to $(GOBIN_DIR)."
