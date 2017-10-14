LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)
BUILD_LDFLAGS := '$(LDFLAGS)'

all: install

checks:
	@echo "Checking deps"
	@(env bash buildscripts/checkdeps.sh)
	@echo "Checking project is in GOPATH"
	@(env bash buildscripts/checkgopath.sh)

getdeps: checks
	@echo "Installing golint" && go get github.com/golang/lint/golint
	@echo "Installing gocyclo" && go get github.com/fzipp/gocyclo
	@echo "Installing deadcode" && go get github.com/remyoudompheng/go-misc/deadcode
	@echo "Installing misspell" && go get github.com/client9/misspell/cmd/misspell

verifiers: vet fmt lint cyclo deadcode spelling

vet:
	@echo "Running $@"
	@go tool vet -all *.go
	@go tool vet -all ./cmd
	@go tool vet -all ./pkg
	@go tool vet -shadow=true *.go
	@go tool vet -shadow=true ./cmd
	@go tool vet -shadow=true ./pkg

spelling:
	@${GOPATH}/bin/misspell *.go
	@${GOPATH}/bin/misspell cmd/*
	@${GOPATH}/bin/misspell pkg/**/*

fmt:
	@echo "Running $@"
	@gofmt -d *.go
	@gofmt -d cmd
	@gofmt -d pkg
lint:
	@echo "Running $@"
	@$(GOPATH)/bin/golint .
	@$(GOPATH)/bin/golint github.com/minio/mc/cmd...
	@$(GOPATH)/bin/golint github.com/minio/mc/pkg...

cyclo:
	@echo "Running $@"
	@$(GOPATH)/bin/gocyclo -over 40 cmd
	@$(GOPATH)/bin/gocyclo -over 40 pkg

deadcode:
	@echo "Running $@"
	@$(GOPATH)/bin/deadcode

build: getdeps verifiers
	@echo "Running $@"
	@go build -tags kqueue --ldflags $(BUILD_LDFLAGS)

test: build
	@echo "Running unit tests"
	@go test $(GOFLAGS) -tags kqueue github.com/minio/mc/cmd...
	@go test $(GOFLAGS) -tags kqueue github.com/minio/mc/pkg...
	@echo "Running functional tests"
	@(env bash $(PWD)/functional-tests.sh)

gomake-all: build
	@echo "Installing mc at $(GOPATH)/bin/mc"
	@cp -af mc $(GOPATH)/bin/mc
	@mkdir -p $(HOME)/.mc

coverage: getdeps verifiers
	@echo "Running all coverage for mc"
	@./buildscripts/go-coverage.sh

pkg-validate-arg-%: ;
ifndef PKG
	$(error Usage: make $(@:pkg-validate-arg-%=pkg-%) PKG=pkg_name)
endif

pkg-add: pkg-validate-arg-add
	@echo "Adding new package $(PKG)"
	@$(GOPATH)/bin/govendor add $(PKG)

pkg-update: pkg-validate-arg-update
	@echo "Updating new package $(PKG)"
	@$(GOPATH)/bin/govendor update $(PKG)

pkg-remove: pkg-validate-arg-remove
	@echo "Remove new package $(PKG)"
	@$(GOPATH)/bin/govendor remove $(PKG)

pkg-list:
	@$(GOPATH)/bin/govendor list

install: gomake-all

release: test
	@MC_RELEASE=RELEASE ./buildscripts/build.sh

experimental: verifiers
	@MC_RELEASE=EXPERIMENTAL ./buildscripts/build.sh

clean:
	@echo "Cleaning up all the generated files"
	@rm -f cover.out
	@rm -f mc
	@find . -name '*.test' | xargs rm -fv
	@rm -fr release
