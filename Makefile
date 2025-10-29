PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)

TARGET_GOARCH ?= $(shell go env GOARCH)
TARGET_GOOS ?= $(shell go env GOOS)

VERSION ?= $(shell git describe --tags)
TAG ?= "minio/mc:$(VERSION)"

GOLANGCI = $(GOPATH)/bin/golangci-lint

all: build

checks:
	@echo "Checking dependencies"
	@(env bash $(PWD)/buildscripts/checkdeps.sh)

getdeps:
	@mkdir -p ${GOPATH}/bin
	@echo "Installing tools" && go install tool

crosscompile:
	@(env bash $(PWD)/buildscripts/cross-compile.sh)

verifiers: getdeps vet lint

docker: build
	@docker build -t $(TAG) . -f Dockerfile.dev

vet:
	@echo "Running $@"
	@GO111MODULE=on go vet github.com/minio/mc/...

lint-fix: getdeps ## runs golangci-lint suite of linters with automatic fixes
	@echo "Running $@ check"
	@$(GOLANGCI) run --build-tags kqueue --timeout=10m --config ./.golangci.yml --fix

lint: getdeps
	@echo "Running $@ check"
	@$(GOLANGCI) run --build-tags kqueue --timeout=10m --config ./.golangci.yml

# Builds mc, runs the verifiers then runs the tests.
check: test
test: verifiers build
	@echo "Running unit tests"
	@GO111MODULE=on CGO_ENABLED=0 go test -tags kqueue ./... 1>/dev/null
	@echo "Running functional tests"
	@GO111MODULE=on MC_TEST_RUN_FULL_SUITE=true go test -race -v --timeout 20m ./... -run Test_FullSuite

test-race: verifiers build
	@echo "Running unit tests under -race"
	@GO111MODULE=on go test -race -v --timeout 20m ./... 1>/dev/null

# Verify mc binary
verify:
	@echo "Verifying build with race"
	@GO111MODULE=on CGO_ENABLED=1 go build -race -tags kqueue -trimpath --ldflags "$(LDFLAGS)" -o $(PWD)/mc 1>/dev/null
	@echo "Running functional tests"
	@GO111MODULE=on MC_TEST_RUN_FULL_SUITE=true go test -race -v --timeout 20m ./... -run Test_FullSuite

# Builds mc locally.
build: checks
	@echo "Building mc binary to './mc'"
	@GO111MODULE=on GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) CGO_ENABLED=0 go build -trimpath -tags kqueue --ldflags "$(LDFLAGS)" -o $(PWD)/mc

hotfix-vars:
	$(eval LDFLAGS := $(shell MC_RELEASE="RELEASE" MC_HOTFIX="hotfix.$(shell git rev-parse --short HEAD)" go run buildscripts/gen-ldflags.go $(shell git describe --tags --abbrev=0 | \
    sed 's#RELEASE\.\([0-9]\+\)-\([0-9]\+\)-\([0-9]\+\)T\([0-9]\+\)-\([0-9]\+\)-\([0-9]\+\)Z#\1-\2-\3T\4:\5:\6Z#')))
	$(eval VERSION := $(shell git describe --tags --abbrev=0).hotfix.$(shell git rev-parse --short HEAD))
	$(eval TAG := "minio/mc:$(VERSION)")

hotfix: hotfix-vars install ## builds mc binary with hotfix tags
	@mv -f ./mc ./mc.$(VERSION)
	@minisign -qQSm ./mc.$(VERSION) -s "${CRED_DIR}/minisign.key" < "${CRED_DIR}/minisign-passphrase"
	@sha256sum < ./mc.$(VERSION) | sed 's, -,mc.$(VERSION),g' > mc.$(VERSION).sha256sum

hotfix-push: hotfix
	@scp -q -r mc.$(VERSION)* minio@dl-0.min.io:~/releases/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/
	@scp -q -r mc.$(VERSION)* minio@dl-1.min.io:~/releases/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/
	@echo "Published new hotfix binaries at https://dl.min.io/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/mc.$(VERSION)"

docker-hotfix-push: docker-hotfix
	@docker push -q $(TAG) && echo "Published new container $(TAG)"

docker-hotfix: hotfix-push checks ## builds mc docker container with hotfix tags
	@echo "Building mc docker image '$(TAG)'"
	@docker build -q --no-cache -t $(TAG) --build-arg RELEASE=$(VERSION) . -f Dockerfile.hotfix

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
