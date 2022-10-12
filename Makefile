goimports := golang.org/x/tools/cmd/goimports@v0.1.12
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0

.PHONY: build.testdata
build.testdata:
	@$(MAKE) build.wat

wat_sources := $(wildcard internal/test/testdata/*/*.wat)
build.wat: $(wat_sources)
	@for f in $^; do \
	    wat2wasm -o $$(echo $$f | sed -e 's/\.wat/\.wasm/') --debug-names $$f; \
	done

.PHONY: test
test:
	@go test -v ./...
	@cd handler/mosn && go test -v ./...

golangci_lint_path := $(shell go env GOPATH)/bin/golangci-lint

$(golangci_lint_path):
	@go install $(golangci_lint)

.PHONY: lint
lint: $(golangci_lint_path)
	@CGO_ENABLED=0 $(golangci_lint_path) run --timeout 5m

.PHONY: format
format:
	@find . -type f -name '*.go' | xargs gofmt -s -w
	@for f in `find . -name '*.go'`; do \
	    awk '/^import \($$/,/^\)$$/{if($$0=="")next}{print}' $$f > /tmp/fmt; \
	    mv /tmp/fmt $$f; \
	done
	@go run $(goimports) -w -local github.com/http-wasm/http-wasm-host-go `find . -name '*.go'`

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
