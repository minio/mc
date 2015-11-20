LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)
BUILD_LDFLAGS := '$(LDFLAGS)'

all: install

checkdeps:
	@echo "Checking deps:"
	@(env bash $(PWD)/buildscripts/checkdeps.sh)

checkgopath:
	@echo "Checking if project is at ${GOPATH}"
	@for mcpath in $(echo ${GOPATH} | sed 's/:/\n/g'); do if [ ! -d ${mcpath}/src/github.com/minio/mc ]; then echo "Project not found in ${mcpath}, please follow instructions provided at https://github.com/minio/minio/blob/master/CONTRIBUTING.md#setup-your-minio-github-repository" && exit 1; fi done
	@echo "BUILD_LDFLAGS: ${BUILD_LDFLAGS}"

getdeps: checkdeps checkgopath
	@go get github.com/golang/lint/golint && echo "Installed golint:"
	@go get golang.org/x/tools/cmd/vet && echo "Installed vet:"
	@go get github.com/fzipp/gocyclo && echo "Installed gocyclo:"
	@go get github.com/remyoudompheng/go-misc/deadcode && echo "Installed deadcode:"

# verifiers: getdeps vet fmt lint cyclo deadcode
verifiers: getdeps vet fmt lint cyclo deadcode

vet:
	@echo "Running $@:"
	@GO15VENDOREXPERIMENT=1 go tool vet -all *.go
	@GO15VENDOREXPERIMENT=1 go tool vet -all ./pkg
	@GO15VENDOREXPERIMENT=1 go tool vet -shadow=true *.go
	@GO15VENDOREXPERIMENT=1 go tool vet -shadow=true ./pkg

fmt:
	@echo "Running $@:"
	@GO15VENDOREXPERIMENT=1 gofmt -s -l *.go
	@GO15VENDOREXPERIMENT=1 gofmt -s -l pkg
lint:
	@echo "Running $@:"
	@GO15VENDOREXPERIMENT=1 golint .
	@GO15VENDOREXPERIMENT=1 golint github.com/minio/mc/pkg...

cyclo:
	@echo "Running $@:"
	@GO15VENDOREXPERIMENT=1 gocyclo -over 30 .

deadcode:
	@echo "Running $@:"
	@GO15VENDOREXPERIMENT=1 deadcode

build: verifiers
	@echo "Installing mc:"

test: verifiers
	@echo "Running all testing:"
	@GO15VENDOREXPERIMENT=1 go test $(GOFLAGS) ./
	@GO15VENDOREXPERIMENT=1 go test $(GOFLAGS) github.com/minio/mc/pkg...

gomake-all: build
	@GO15VENDOREXPERIMENT=1 go build --ldflags $(BUILD_LDFLAGS) -o $(GOPATH)/bin/mc
	@mkdir -p $(HOME)/.mc

coverage:
	@GO15VENDOREXPERIMENT=1 go test -race -coverprofile=cover.out ./
	@go tool cover -html=cover.out && echo "Visit your browser"

pkg-add:
	@GO15VENDOREXPERIMENT=1 govendor add $(PKG)

pkg-update:
	@GO15VENDOREXPERIMENT=1 govendor update $(PKG)

pkg-remove:
	@GO15VENDOREXPERIMENT=1 govendor remove $(PKG)

install: gomake-all

clean:
	@rm -fv cover.out
	@rm -fv mc
