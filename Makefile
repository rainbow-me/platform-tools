# Project Metadata
# These act as inputs to other makefile targets.
#
ORG := rainbow
PROJECT := platfomt-tools
VERSION := $(shell git describe --tags --always 2>/dev/null || git rev-parse --short HEAD)
BUILD := $(shell git rev-parse --short HEAD)
DOCKER_TAG="rainbow/$(PROJECT):$(VERSION)"


#
# Tool Prerequisites Check
# This ensures that you have the necessary executables installed to run this makefile.
#
BUILD_PREREQUISITES = git go
VALIDATION_PREREQUISITES = golangci-lint gci buf protoc-gen-go protoc-gen-go-grp

# modules
MODULES := common grpc

#
# Build Options
# Typical inputs to the build targets found below.
#
TARGET=target
BIN=$(TARGET)/bin
LDFLAGS=-ldflags "-X=main.Version=$(VERSION) -X=main.Build=$(BUILD)"


.PHONY: usage
usage:
	@ echo "Usage: make [`cat Makefile | grep "^[A-z\%\-]*:" | awk '{print $$1}' | sed "s/://g" | sed "s/%/[1-3]/g" | xargs`]"


.PHONY: info
info:
	@ echo ORG: $(ORG)
	@ echo PROJECT: $(PROJECT)
	@ echo VERSION: $(VERSION)
	@ echo BUILD: $(BUILD)
	@ echo DOCKER_TAG: $(DOCKER_TAG)



.PHONY: clean
clean: info tidy


.PHONY: tidy
tidy:
	@echo "Running go mod tidy in all modules..."
	@for mod in $(MODULES); do \
		echo "Tidying $$mod/"; \
		(cd $$mod && go mod tidy && git diff --exit-code -- go.mod go.sum) || exit 1; \
	done

.PHONY: validation_deps
validation_deps: info clean
	@ printf $(foreach exec,$(VALIDATION_PREREQUISITES), \
        $(if $(shell which $(exec)),"", \
        $(error "No $(exec) in PATH. Prerequisites are: $(VALIDATION_PREREQUISITES)")))

 .PHONY: install_tools
 install_tools:
	@echo "--- Installing tools..."
	@sh scripts/install-go-tools.sh


# Installs project dependencies.
.PHONY: install
install: info clean
	@echo "--- Installing project dependencies..."
	@GOPRIVATE=github.com/rainbow-me go get ./...

# lint app
.PHONY: lint
lint:
	@echo "Running golangci-lint on all modules..."
	@for mod in $(MODULES); do \
		echo "Linting $$mod/"; \
		(cd $$mod && golangci-lint run --timeout=3m --config=../.golangci.yaml ./...) || exit 1; \
	done


.PHONY: fmt
fmt:
	@gofmt -w .
	@gci write --skip-generated -s standard -s default -s "prefix(github.com/rainbow-me)" .

# Run tests with the race detector both enabled and disabled. Enabling the race
# detector can affect timing of events which can mask non-data race failures.
# Additionally, cgo builds are not entirely reflective of production builds
# without cgo enabled (separate DNS resolvers, dynamic linking behavior, etc).
.PHONY: unit
unit:
	@echo "🐢 Running tests without race detector..."
	@for mod in $(MODULES); do \
		echo "▶ Running tests in $$mod (no race detector)"; \
		( cd $$mod && CGO_ENABLED=0 gotestsum --format testname --junitfile ../junit-tests-$$mod.xml -- -cover -coverprofile=../coverage-$$mod.out ./... ); \
	done

	@echo "🏎️ Running tests with race detector..."
	@for mod in $(MODULES); do \
		echo "▶ Running tests in $$mod (with race detector)"; \
		( cd $$mod && CGO_ENABLED=1 GOFLAGS="-ldflags=-extldflags=-Wl,-ld_classic" gotestsum --format testname -- -race ./... ); \
	done

test-setup:
	@which gotestsum 2>&1 > /dev/null || go install gotest.tools/gotestsum@latest

.PHONY: coverage
coverage:
	@echo "Generating merged coverage report..."
	@cat coverage-*.out | grep -v "mode:" | awk 'BEGIN { print "mode: set" } 1' > coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out

.PHONY: test
test: info clean  test-setup unit coverage


.PHONY: govulncheck
govulncheck:
	@echo "Running govulncheck for each module..."
	@for mod in $(MODULES); do \
		if ! command -v $$HOME/go/bin/govulncheck &> /dev/null; then \
			echo "Installing govulncheck..."; \
			GO111MODULE=on go install golang.org/x/vuln/cmd/govulncheck@latest; \
		fi; \
		echo "Checking module: $$mod"; \
		(cd $$mod && govulncheck ./...); \
	done


.PHONY: proto_lint
proto_lint: info clean
	@ printf "\nLint Protos\n"
	@ cd grpc && buf lint


.PHONY: gen_deps
gen_deps: info
	@ printf "\nDeps protos\n"
	@ cd grpc/protos/v1 && buf dep update


.PHONY: gen
gen: info
	@ cd grpc/protos && pwd && buf generate v1 --template v1/buf.gen.yaml