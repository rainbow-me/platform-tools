# Project Metadata
# These act as inputs to other makefile targets.
#
ORG := rainbow
PROJECT := platform-tools
VERSION := $(shell git describe --tags --always 2>/dev/null || git rev-parse --short HEAD)
BUILD := $(shell git rev-parse --short HEAD)
DOCKER_TAG="rainbow/$(PROJECT):$(VERSION)"
GO_TARGETS=./...


#
# Tool Prerequisites Check
# This ensures that you have the necessary executables installed to run this makefile.
#
BUILD_PREREQUISITES = git go
VALIDATION_PREREQUISITES = golangci-lint gci buf protoc-gen-go protoc-gen-go-grp

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
	@echo "Running go mod tidy"
	@go mod tidy && git diff --exit-code -- go.mod go.sum || exit 1

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
	@GOPRIVATE=github.com/rainbow-me go get ${GO_TARGETS}

# lint app
.PHONY: lint
lint:
	@echo "Running golangci-lint"
	@golangci-lint run ${GO_TARGETS} || exit 1


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
	CGO_ENABLED=0 gotestsum --format testname --junitfile ../junit-tests-$$mod.xml -- -cover -coverprofile=../coverage-$$mod.out ${GO_TARGETS}

	@echo "🏎️ Running tests with race detector..."
	CGO_ENABLED=1 gotestsum --format testname -- -race ${GO_TARGETS}

test-setup:
	@which gotestsum 2>&1 > /dev/null || go install gotest.tools/gotestsum@latest

.PHONY: coverage
coverage:
	@echo "Generating merged coverage report..."
	@cat coverage-*.out | grep -v "mode:" | awk 'BEGIN { print "mode: set" } 1' > coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out

.PHONY: test
test: info test-setup unit coverage

.PHONY: test/fast
test/fast:
	go test ./...

.PHONY: govulncheck
govulncheck:
	@echo "Running govulncheck"
	@if ! command -v $$HOME/go/bin/govulncheck &> /dev/null; then \
		echo "Installing govulncheck..."; \
		GO111MODULE=on go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi;
	@govulncheck ${GO_TARGETS}


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