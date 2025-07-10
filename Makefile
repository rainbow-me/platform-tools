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
VALIDATION_PREREQUISITES = golangci-lint gci

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

.PHONY: clean
clean: info
	@ rm -rf target
	@ go mod tidy


.PHONY: tidy tidy-common tidy-grpc
tidy: tidy-common tidy-grpc
tidy-common:
	@echo "Running go mod tidy in common/"
	@(cd common && go mod tidy && git diff --exit-code -- go.mod go.sum)

tidy-grpc:
	@echo "Running go mod tidy in grpc/"
	@(cd grpc && go mod tidy && git diff --exit-code -- go.mod go.sum)

.PHONY: validation_deps
validation_deps: info clean
	@ printf $(foreach exec,$(VALIDATION_PREREQUISITES), \
        $(if $(shell which $(exec)),"", \
        $(error "No $(exec) in PATH. Prerequisites are: $(VALIDATION_PREREQUISITES)")))

.PHONY: build_deps
build_deps: info clean
	@ printf $(foreach exec,$(BUILD_PREREQUISITES), \
        $(if $(shell which $(exec)),"", \
        $(error "No $(exec) in PATH. Prerequisites are: $(BUILD_PREREQUISITES)")))


.PHONY: info
info:
	@ echo ORG: $(ORG)
	@ echo PROJECT: $(PROJECT)
	@ echo VERSION: $(VERSION)
	@ echo BUILD: $(BUILD)
	@ echo DOCKER_TAG: $(DOCKER_TAG)


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
lint: lint-common lint-grpc

lint-common:
	@echo "Linting common/"
	@(cd common && golangci-lint run --timeout=3m --config=../.golangci.yaml ./...)

lint-grpc:
	@echo "Linting grpc/"
	@(cd grpc && golangci-lint run --timeout=3m --config=../.golangci.yaml ./...)


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
	@ CGO_ENABLED=0 gotestsum --format testname --junitfile junit-tests.xml -- -cover -coverprofile=coverage.out ./...

	@echo "🏎️ Running tests with race detector..."
	@ CGO_ENABLED=1 gotestsum --format testname -- -race  ./...

test-setup:
	@which gotestsum 2>&1 > /dev/null || go install gotest.tools/gotestsum@latest

.PHONY: coverage
coverage:
	@go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

.PHONY: test
test: info clean  test-setup unit coverage


.PHONY: govulncheck
govulncheck:
	@if ! command -v ${HOME}/go/bin/govulncheck &> /dev/null; then \
		GO111MODULE=on go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...

.PHONY: build
build: info build_deps
	@ printf "\nBuild app\n"
	@ go build $(LDFLAGS) -o $(BIN)/$(PROJECT) cmd/main.go
	@ cp -R cmd/config $(BIN)/