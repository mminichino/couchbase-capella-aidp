BINARY_NAME=terraform-provider-couchbase-capella-aidp
DESTINATION=./bin/$(BINARY_NAME)

GITTAG=$(shell git describe --tags --abbrev=0 2>/dev/null)
VERSION=$(GITTAG:v%=%)
ifeq ($(VERSION),)
VERSION=devel
endif
LINKER_FLAGS=-s -w -X 'github.com/mminichino/couchbase-capella-aidp/version.ProviderVersion=$(VERSION)'

GOFMT_FILES?=$$(find . -name '*.go')
TFPLUGINDOCS_VERSION=v0.22.0

default: build

.PHONY: help
help: ## Show available commands
	@grep -h -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the provider binary
	@go build -ldflags "$(LINKER_FLAGS)" -o $(DESTINATION)

.PHONY: install
install: build ## Install the provider into the local Terraform plugins directory
	@mkdir -p ~/.terraform.d/plugins/registry.terraform.io/mminichino/couchbase-capella-aidp/$(VERSION)/$$(go env GOOS)_$$(go env GOARCH)
	@cp $(DESTINATION) ~/.terraform.d/plugins/registry.terraform.io/mminichino/couchbase-capella-aidp/$(VERSION)/$$(go env GOOS)_$$(go env GOARCH)/

.PHONY: fmt
fmt: ## Format Go code
	@gofmt -s -w $(GOFMT_FILES)

.PHONY: vet
vet: ## Run go vet
	@go vet ./...

.PHONY: test
test: ## Run unit tests
	@go test ./... -count=1

.PHONY: testacc
testacc: ## Run acceptance tests (requires TF_ACC=1 and Capella credentials)
	TF_ACC=1 go test ./tests -v -count=1 -timeout 120m

.PHONY: generate-docs
generate-docs: ## Generate provider documentation with tfplugindocs
	@go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION) generate \
		--provider-dir . \
		--provider-name couchbase-capella-aidp \
		--rendered-provider-name couchbase-capella-aidp \
		--examples-dir ./examples \
		--rendered-website-dir ./docs

.PHONY: release-snapshot
release-snapshot: ## Build a local GoReleaser snapshot (no publish)
	goreleaser release --snapshot --clean --skip=publish,sign

.PHONY: check
check: fmt vet test ## Format, vet, and test

.PHONY: tidy
tidy: ## Tidy go modules
	@go mod tidy
