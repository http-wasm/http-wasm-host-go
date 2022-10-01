goimports := golang.org/x/tools/cmd/goimports@v0.1.12
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0

.PHONY: test
test:
	@go test -v ./...
	@(cd handler/fasthttp; go test -v ./...)

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
	@(cd handler/fasthttp; go mod tidy)
	@if [ ! -z "`git status -s`" ]; then \
		echo "The following differences will fail CI until committed:"; \
		git diff --exit-code; \
	fi

.PHONY: clean
clean: ## Ensure a clean build
	@go clean -testcache
