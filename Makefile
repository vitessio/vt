.PHONY: all build test tidy clean vitess-tester vtbenchstat

GO := go
REQUIRED_GO_VERSION := 1.23

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

build: vitess-tester vtbenchstat

vitess-tester:
	$(GO) build -o $@ ./

vtbenchstat:
	$(GO) build -o $@ ./src/cmd/vtbenchstat

test: build
	$(GO) test -cover ./...
	#./vitess-tester -check-error

tidy:
	$(GO) mod tidy

clean:
	$(GO) clean -i ./...
	rm -f vitess-tester vtbenchstat