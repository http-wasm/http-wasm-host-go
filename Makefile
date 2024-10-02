gofumpt := mvdan.cc/gofumpt@v0.5.0
gosimports := github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.1

.PHONY: testdata
testdata:
	@$(MAKE) build.wat

wat_sources := $(wildcard examples/*.wat) $(wildcard internal/test/testdata/*/*.wat)
build.wat: $(wat_sources)
	@for f in $^; do \
        wasm=$$(echo $$f | sed -e 's/\.wat/\.wasm/'); \
	    wat2wasm -o $$wasm --debug-names $$f; \
	done

.PHONY: test
test:
	@go test -v ./...

.PHONY: bench
bench:
	@(cd handler/nethttp; go test -run=NONE -bench=. .)

golangci_lint_path := $(shell go env GOPATH)/bin/golangci-lint

$(golangci_lint_path):
	@go install $(golangci_lint)

.PHONY: lint
lint: $(golangci_lint_path)
	@CGO_ENABLED=0 $(golangci_lint_path) run --timeout 5m

.PHONY: format
format:
	@go run $(gofumpt) -l -w .
	@go run $(gosimports) -local github.com/http-wasm/ -w $(shell find . -name '*.go' -type f)

.PHONY: check
check:
	@$(MAKE) lint
	@$(MAKE) format
	@go mod tidy
	@if [ ! -z "`git status -s`" ]; then \
		echo "The following differences will fail CI until committed:"; \
		git diff --exit-code; \
	fi

.PHONY: clean
clean: ## Ensure a clean build
	@go clean -testcache

# note: the guest wasm is stored in tck/, not tck/guest, so that go:embed can read it.
.PHONY: tck
tck:
	@cd tck/guest && tinygo build -o ../tck.wasm -scheduler=none --no-debug -target=wasi .
