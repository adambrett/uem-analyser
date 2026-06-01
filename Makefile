export

SHELL := /bin/bash -o errexit -o nounset -o pipefail

MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

VERBOSE ?= false
ifeq (${VERBOSE}, false)
	MAKEFLAGS += --silent
endif

# Variables
GOBASE ?= ${CURDIR}
GOBIN  := ${GOBASE}/bin

DIST_DIR              := dist
GOCACHE               ?= ${CURDIR}/.cache/go-build
GOLANGCI_LINT_CACHE   ?= ${CURDIR}/.cache/golangci-lint
PORT                  ?= 8080
WASM_BINARY           := ${DIST_DIR}/app.wasm
WEB_DIR               := web
COVER_MIN             ?= 80
COVERPROFILE          ?= coverage.out
COVER_PACKAGES        ?= ./pkg/parser ./internal/analyser ./internal/spreadsheet
GOFUMPT_VERSION       ?= v0.5.0
GOIMPORTS_REVISER_VERSION ?= v3.4.1
GOLANGCI_LINT_VERSION ?= v2.4.0

# Ensure that we use local project binaries before consulting the system.
PATH := ${GOBIN}:${PATH}

# Applications
GO     ?= go
PYTHON ?= python3

GOLANGCI_LINT     ?= ${GOBIN}/golangci-lint
GOFUMPT           ?= ${GOBIN}/gofumpt
GOIMPORTS_REVISER ?= ${GOBIN}/goimports-reviser

# Helpers
.PHONY: all
all: test build ## Run tests and build the static site

.PHONY: run
run: serve ## Alias for serve

.PHONY: serve
serve: build ## Serve the built site locally
	$(PYTHON) -m http.server ${PORT} --directory ${DIST_DIR}

.PHONY: depend
depend: ## Update project dependencies
	$(GO) mod tidy

$(GOFUMPT):
	$(GO) install mvdan.cc/gofumpt@${GOFUMPT_VERSION}

$(GOIMPORTS_REVISER):
	$(GO) install github.com/incu6us/goimports-reviser/v3@${GOIMPORTS_REVISER_VERSION}

$(GOLANGCI_LINT):
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

.PHONY: fmt
fmt: $(GOIMPORTS_REVISER) $(GOFUMPT) ## Format Go files
	find . -type f -name '*.go' -not -path "./vendor/*" | \
		xargs -I {} $(GOIMPORTS_REVISER) -company-prefixes="github.com/adambrett/" -project-name="github.com/adambrett/uem-analyser" {}
	# In some cases you need to run gofumpt twice to resolve all formatting issues as one simplification
	# can allow another one, but gofumpt is not smart enough to apply both at the same time.
	find . -type f -name '*.go' -not -path "./vendor/*" | xargs $(GOFUMPT) -w
	find . -type f -name '*.go' -not -path "./vendor/*" | xargs $(GOFUMPT) -w

.PHONY: fmt-check
fmt-check: $(GOIMPORTS_REVISER) $(GOFUMPT) ## Check Go file formatting
	set +e; \
	imports_output="$$(find . -type f -name '*.go' -not -path "./vendor/*" | \
		xargs -I {} $(GOIMPORTS_REVISER) -company-prefixes="github.com/adambrett/" -project-name="github.com/adambrett/uem-analyser" -list-diff -set-exit-status {} 2>&1)"; \
	imports_status=$$?; \
	set -e; \
	if [[ $${imports_status} -ne 0 ]]; then \
		printf 'The following files need import cleanup:\n%s\n' "$${imports_output}"; \
		printf "Run 'make fmt' to fix.\n"; \
		exit 1; \
	fi
	output="$$(find . -type f -name '*.go' -not -path "./vendor/*" | xargs $(GOFUMPT) -l)"; \
	if [[ -n "$${output}" ]]; then \
		printf 'The following files are not gofumpt-clean:\n%s\n' "$${output}"; \
		printf "Run 'make fmt' to fix.\n"; \
		exit 1; \
	fi

# Linting
.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run the linter
	$(GOLANGCI_LINT) run ./...

# Testing
.PHONY: test
test: ## Run Go tests with the race detector
	$(GO) test -race ./...

.PHONY: coverage
coverage: ## Run coverage and require total coverage to meet COVER_MIN
	$(GO) test -race -coverprofile=${COVERPROFILE} ${COVER_PACKAGES}
	$(GO) tool cover -func=${COVERPROFILE} | awk -v min="${COVER_MIN}" '{ print } /^total:/ { pct = $$3; sub(/%/, "", pct); if (pct < min) { printf "coverage %.1f%% is below %.1f%%\n", pct, min; exit 1 } }'

.PHONY: test-wasm
test-wasm: ## Run package tests under the browser WASM target
	GOOS=js GOARCH=wasm $(GO) test -exec="$$($(GO) env GOROOT)/lib/wasm/go_js_wasm_exec" ./pkg/parser ./internal/analyser ./internal/spreadsheet

# Building
.PHONY: build
build: ## Build the deployable static site
	rm -rf ${DIST_DIR}
	mkdir -p ${DIST_DIR}
	GOOS=js GOARCH=wasm $(GO) build -o ${WASM_BINARY} ./cmd/uem-analyser
	cp "$$($(GO) env GOROOT)/lib/wasm/wasm_exec.js" ${DIST_DIR}/wasm_exec.js
	cp ${WEB_DIR}/index.html ${DIST_DIR}/index.html
	cp ${WEB_DIR}/app.css ${DIST_DIR}/app.css
	cp ${WEB_DIR}/app.js ${DIST_DIR}/app.js
	cp ${WEB_DIR}/CNAME ${DIST_DIR}/CNAME
	cp -R ${WEB_DIR}/vendor ${DIST_DIR}/vendor
	touch ${DIST_DIR}/.nojekyll

# Cleaning
.PHONY: clean
clean: ## Remove build artifacts
	rm -rf ${DIST_DIR}
	rm -f ${COVERPROFILE}

# Make Helpers
.PHONY: help
help: ## Print this help message
	grep -E '^[/a-zA-Z_-]+:.*?## .*$$' ${MAKEFILE_LIST} | sort | awk 'BEGIN {FS = ":|##"}; {printf "%-20s\033[36m%-20s \033[0m %s\n", $$1, $$2, $$4}'

print-%: ## Print the value of a variable
	echo $* = $($*)
