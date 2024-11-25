.PHONY: all build test tidy clean pretty install-tools lint install-hooks

GO := go
REQUIRED_GO_VERSION := 1.23
GOLANGCI_LINT_VERSION := v1.62.0

# Version check
check_version:
	@GO_VERSION=$$($(GO) version | awk '{print $$3}' | sed 's/go//'); \
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
	$(GO) build -o vt ./go/vt

test:
	$(GO) test -count=1 ./go/...

tidy:
	$(GO) mod tidy

clean:
	$(GO) clean -i ./...
	rm -f vt

# Pretty: formats the code using gofumpt and goimports-reviser
pretty: check-tools
	@echo "Running formatting tools..."
	@gofumpt -l -w . >/dev/null 2>&1 || true
	@goimports-reviser -project-name $$(go list -m) -rm-unused -set-alias -format . >/dev/null 2>&1 || true

# Tools installation command
install-tools:
	@echo "Installing gofumpt..."
	go install mvdan.cc/gofumpt@latest

	@echo "Installing goimports-reviser..."
	go install github.com/incu6us/goimports-reviser/v3@latest

	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)

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