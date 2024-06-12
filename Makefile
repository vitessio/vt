.PHONY: all build test tidy clean

GO := go

default: build

build:
	$(GO) build -o vitess-tester ./

test: build
	$(GO) test -cover ./...
	#./vitess-tester -check-error

tidy:
	$(GO) mod tidy

clean:
	$(GO) clean -i ./...
	rm -rf vitess-tester
