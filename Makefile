PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)

BUILD_LDFLAGS := '$(LDFLAGS)'

all: build

checks:
	@echo "Checking dependencies"
	@(env bash $(PWD)/buildscripts/checkdeps.sh)

getdeps:
	@echo "Installing golint" && go get golang.org/x/lint/golint
	@echo "Installing gocyclo" && go get github.com/fzipp/gocyclo
	@echo "Installing misspell" && go get github.com/client9/misspell/cmd/misspell
	@echo "Installing ineffassign" && go get github.com/gordonklaus/ineffassign

verifiers: getdeps vet fmt lint cyclo spelling

vet:
	@echo "Running $@"
	@go vet github.com/minio/mc/...

fmt:
	@echo "Running $@"
	@gofmt -d cmd
	@gofmt -d pkg

lint:
	@echo "Running $@"
	@${GOPATH}/bin/golint -set_exit_status github.com/minio/mc/cmd...
	@${GOPATH}/bin/golint -set_exit_status github.com/minio/mc/pkg...

ineffassign:
	@echo "Running $@"
	@${GOPATH}/bin/ineffassign .

cyclo:
	@echo "Running $@"
	@${GOPATH}/bin/gocyclo -over 100 cmd
	@${GOPATH}/bin/gocyclo -over 100 pkg

spelling:
	@${GOPATH}/bin/misspell -error `find cmd/`
	@${GOPATH}/bin/misspell -error `find pkg/`
	@${GOPATH}/bin/misspell -error `find docs/`

# Builds MinIO, runs the verifiers then runs the tests.
check: test
test: verifiers build
	@echo "Running unit tests"
	@GO111MODULE=on CGO_ENABLED=0 go test -tags kqueue ./...
	@echo "Running functional tests"
	@(env bash $(PWD)/functional-tests.sh)

coverage: build
	@echo "Running all coverage for MinIO"
	@(env bash $(PWD)/buildscripts/go-coverage.sh)

# Builds MinIO locally.
build: checks
	@echo "Building MinIO binary to './mc'"
	@GO111MODULE=on GO_FLAGS="" CGO_ENABLED=0 go build -tags kqueue --ldflags $(BUILD_LDFLAGS) -o $(PWD)/mc

# Builds MinIO and installs it to $GOPATH/bin.
install: build
	@echo "Installing mc binary to '$(GOPATH)/bin/mc'"
	@mkdir -p $(GOPATH)/bin && cp -uf $(PWD)/mc $(GOPATH)/bin/mc
	@echo "Installation successful. To learn more, try \"mc --help\"."

clean:
	@echo "Cleaning up all the generated files"
	@find . -name '*.test' | xargs rm -fv
	@find . -name '*~' | xargs rm -fv
	@rm -rvf mc
	@rm -rvf build
	@rm -rvf release
