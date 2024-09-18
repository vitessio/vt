.PHONY: all build test tidy clean vitess-tester vtbenchstat

GO := go

default: build

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