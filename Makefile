LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)
BUILD_LDFLAGS := '$(LDFLAGS) -s -w'

all: install

checks:
	@echo "Checking deps:"
	@(env bash buildscripts/checkdeps.sh)
	@(env bash buildscripts/checkgopath.sh)

getdeps: checks
	@go get github.com/golang/lint/golint && echo "Installed golint:"
	@go get github.com/fzipp/gocyclo && echo "Installed gocyclo:"
	@go get github.com/remyoudompheng/go-misc/deadcode && echo "Installed deadcode:"
	@go get github.com/client9/misspell/cmd/misspell && echo "Installed misspell:"

# verifiers: getdeps vet fmt lint cyclo deadcode
verifiers: vet fmt lint cyclo deadcode spelling

vet:
	@echo "Running $@:"
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
	@echo "Running $@:"
	@gofmt -s -l *.go
	@gofmt -s -l cmd
	@gofmt -s -l pkg
lint:
	@echo "Running $@:"
	@$(GOPATH)/bin/golint .
	@$(GOPATH)/bin/golint github.com/minio/mc/cmd...
	@$(GOPATH)/bin/golint github.com/minio/mc/pkg...

cyclo:
	@echo "Running $@:"
	@$(GOPATH)/bin/gocyclo -over 40 cmd
	@$(GOPATH)/bin/gocyclo -over 40 pkg

deadcode:
	@echo "Running $@:"
	@$(GOPATH)/bin/deadcode

build: getdeps verifiers

test: getdeps verifiers
	@echo "Running all testing:"
	@go test $(GOFLAGS) github.com/minio/mc/cmd...
	@go test $(GOFLAGS) github.com/minio/mc/pkg...

gomake-all: build
	@echo "Installing mc:"
	@go build --ldflags $(BUILD_LDFLAGS) -o $(GOPATH)/bin/mc
	@mkdir -p $(HOME)/.mc

coverage: getdeps verifiers
	@echo "Running all coverage:"
	@./buildscripts/go-coverage.sh

pkg-validate-arg-%: ;
ifndef PKG
	$(error Usage: make $(@:pkg-validate-arg-%=pkg-%) PKG=pkg_name)
endif

pkg-add: pkg-validate-arg-add
	@$(GOPATH)/bin/govendor add $(PKG)

pkg-update: pkg-validate-arg-update
	@$(GOPATH)/bin/govendor update $(PKG)

pkg-remove: pkg-validate-arg-remove
	@$(GOPATH)/bin/govendor remove $(PKG)

pkg-list:
	@$(GOPATH)/bin/govendor list

install: gomake-all

all-tests: test
	# TODO disable them for now.
	#@./tests/test-minio.sh

release: verifiers
	@MC_RELEASE=RELEASE ./buildscripts/build.sh

experimental: verifiers
	@MC_RELEASE=EXPERIMENTAL ./buildscripts/build.sh

clean:
	@rm -f cover.out
	@rm -f mc
	@find . -name '*.test' | xargs rm -fv
	@rm -fr release
