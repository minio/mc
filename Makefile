PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)

GOOS := $(shell go env GOOS)
GOOSALT ?= 'linux'
ifeq ($(GOOS),'darwin')
  GOOSALT = 'mac'
endif

BUILD_LDFLAGS := '$(LDFLAGS)'

all: build

checks:
	@echo "Checking dependencies"
	@(env bash $(PWD)/buildscripts/checkdeps.sh)

getdeps:
	@mkdir -p ${GOPATH}/bin
	@which golint 1>/dev/null || (echo "Installing golint" && go get -u golang.org/x/lint/golint)
	@which staticcheck 1>/dev/null || (echo "Installing staticcheck" && wget --quiet https://github.com/dominikh/go-tools/releases/download/2019.2.3/staticcheck_${GOOS}_amd64.tar.gz && tar xf staticcheck_${GOOS}_amd64.tar.gz && mv staticcheck/staticcheck ${GOPATH}/bin/staticcheck && chmod +x ${GOPATH}/bin/staticcheck && rm -f staticcheck_${GOOS}_amd64.tar.gz && rm -rf staticcheck)
	@which misspell 1>/dev/null || (echo "Installing misspell" && wget --quiet https://github.com/client9/misspell/releases/download/v0.3.4/misspell_0.3.4_${GOOSALT}_64bit.tar.gz && tar xf misspell_0.3.4_${GOOSALT}_64bit.tar.gz && mv misspell ${GOPATH}/bin/misspell && chmod +x ${GOPATH}/bin/misspell && rm -f misspell_0.3.4_${GOOSALT}_64bit.tar.gz)

crosscompile:
	@(env bash $(PWD)/buildscripts/cross-compile.sh)

verifiers: getdeps vet fmt lint staticcheck spelling

vet:
	@echo "Running $@"
	@GO111MODULE=on go vet github.com/minio/mc/...

fmt:
	@echo "Running $@"
	@GO111MODULE=on gofmt -d cmd/
	@GO111MODULE=on gofmt -d pkg/

lint:
	@echo "Running $@"
	@GO111MODULE=on ${GOPATH}/bin/golint -set_exit_status github.com/minio/mc/cmd/...
	@GO111MODULE=on ${GOPATH}/bin/golint -set_exit_status github.com/minio/mc/pkg/...

staticcheck:
	@echo "Running $@"
	@GO111MODULE=on ${GOPATH}/bin/staticcheck github.com/minio/mc/cmd/...
	@GO111MODULE=on ${GOPATH}/bin/staticcheck github.com/minio/mc/pkg/...

spelling:
	@GO111MODULE=on ${GOPATH}/bin/misspell -locale US -error `find cmd/`
	@GO111MODULE=on ${GOPATH}/bin/misspell -locale US -error `find pkg/`
	@GO111MODULE=on ${GOPATH}/bin/misspell -locale US -error `find docs/`

# Builds mc, runs the verifiers then runs the tests.
check: test
test: verifiers build
	@echo "Running unit tests"
	@GO111MODULE=on CGO_ENABLED=0 go test -tags kqueue ./... 1>/dev/null
	@echo "Running functional tests"
	@(env bash $(PWD)/functional-tests.sh)

coverage: build
	@echo "Running all coverage for MinIO"
	@(env bash $(PWD)/buildscripts/go-coverage.sh)

# Builds mc locally.
build: checks
	@echo "Building mc binary to './mc'"
	@GO111MODULE=on CGO_ENABLED=0 go build -tags kqueue --ldflags $(BUILD_LDFLAGS) -o $(PWD)/mc

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
