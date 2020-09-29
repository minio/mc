PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)

GOARCH := $(shell go env GOARCH)
GOOS := $(shell go env GOOS)

BUILD_LDFLAGS := '$(LDFLAGS)'

VERSION ?= $(shell git describe --tags)
TAG ?= "minio/mc:$(VERSION)"

all: build

checks:
	@echo "Checking dependencies"
	@(env bash $(PWD)/buildscripts/checkdeps.sh)

getdeps:
	@mkdir -p ${GOPATH}/bin
	@which golangci-lint 1>/dev/null || (echo "Installing golangci-lint" && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.27.0)
	@which msgp 1>/dev/null || (echo "Installing msgp" && go get github.com/tinylib/msgp)
	@which stringer 1>/dev/null || (echo "Installing stringer" && go get golang.org/x/tools/cmd/stringer)

crosscompile:
	@(env bash $(PWD)/buildscripts/cross-compile.sh)

verifiers: getdeps vet fmt lint check-gen

docker: build
	@docker build -t $(TAG) . -f Dockerfile.dev

check-gen:
	@go generate ./... >/dev/null
	@(! git diff --name-only | grep '_gen.go$$') || (echo "Non-committed changes in auto-generated code is detected, please commit them to proceed." && false)

vet:
	@echo "Running $@"
	@GO111MODULE=on go vet github.com/minio/mc/...

fmt:
	@echo "Running $@"
	@GO111MODULE=on gofmt -d cmd/
	@GO111MODULE=on gofmt -d pkg/

lint:
	@echo "Running $@ check"
	@GO111MODULE=on ${GOPATH}/bin/golangci-lint cache clean
	@GO111MODULE=on ${GOPATH}/bin/golangci-lint run --timeout=5m --config ./.golangci.yml

# Builds mc, runs the verifiers then runs the tests.
check: test
test: verifiers build
	@echo "Running unit tests"
	@GO111MODULE=on CGO_ENABLED=0 go test -tags kqueue ./... 1>/dev/null
	@echo "Running functional tests"
	@(env bash $(PWD)/functional-tests.sh)

test-race: verifiers build
	@echo "Running unit tests under -race"
	@GO111MODULE=on go test -race -v --timeout 20m ./... 1>/dev/null

# Verify mc binary
verify:
	@echo "Verifying build with race"
	@GO111MODULE=on CGO_ENABLED=1 go build -race -tags kqueue -trimpath --ldflags "$(LDFLAGS)" -o $(PWD)/mc 1>/dev/null
	@echo "Running functional tests"
	@(env bash $(PWD)/functional-tests.sh)

# Builds mc locally.
build: checks
	@echo "Building mc binary to './mc'"
	@GO111MODULE=on CGO_ENABLED=0 go build -trimpath -tags kqueue --ldflags $(BUILD_LDFLAGS) -o $(PWD)/mc

# Builds MinIO and installs it to $GOPATH/bin.
install: build
	@echo "Installing mc binary to '$(GOPATH)/bin/mc'"
	@mkdir -p $(GOPATH)/bin && cp -f $(PWD)/mc $(GOPATH)/bin/mc
	@echo "Installation successful. To learn more, try \"mc --help\"."

clean:
	@echo "Cleaning up all the generated files"
	@find . -name '*.test' | xargs rm -fv
	@find . -name '*~' | xargs rm -fv
	@rm -rvf mc
	@rm -rvf build
	@rm -rvf release
