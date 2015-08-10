## filter multiple GOPATH
all: getdeps install

checkdeps:
	@echo "Checking deps:"
	@(env bash $(PWD)/buildscripts/checkdeps.sh)

checkgopath:
	@echo "Checking if project is at ${GOPATH}"
	@for mcpath in $(echo ${GOPATH} | sed 's/:/\n/g' | grep -v Godeps); do if [ ! -d ${mcpath}/src/github.com/minio/mc ]; then echo "Project not found in ${mcpath}, please follow instructions provided at https://github.com/minio/minio/blob/master/CONTRIBUTING.md#setup-your-minio-github-repository" && exit 1; fi done

getdeps: checkdeps checkgopath
	@go get github.com/tools/godep && echo "Installed godep:"
	@go get github.com/golang/lint/golint && echo "Installed golint:"
	@go get golang.org/x/tools/cmd/vet && echo "Installed vet:"
	@go get github.com/fzipp/gocyclo && echo "Installed gocyclo:"
	@go get github.com/remyoudompheng/go-misc/deadcode && echo "Installed deadcode:"

# verifiers: getdeps vet fmt lint cyclo deadcode
verifiers: getdeps vet fmt lint cyclo

vet:
	@echo "Running $@:"
	@go vet ./...
fmt:
	@echo "Running $@:"
	@test -z "$$(gofmt -s -l . 2>&1 | grep -v 'Godeps/_workspace/src/')" || \
		echo "+ please format Go code with 'gofmt -s'"
lint:
	@echo "Running $@:"
	@test -z "$$(golint ./... 2>&1 | grep -v 'Godeps/_workspace/src/')"

cyclo:
	@echo "Running $@:"
	@test -z "$$(gocyclo -over 30 . 2>&1 | grep -v 'Godeps/_workspace/src/')"

deadcode:
	@echo "Running $@:"
	@test -z "$$(deadcode 2>&1 | grep -v 'Godeps/_workspace/src/')"

build: getdeps verifiers
	@echo "Installing mc:"
	@godep go test -race ./...

gomake-all: build
	@godep go install github.com/minio/mc
	@mkdir -p $(HOME)/.mc

release: genversion
	@echo "Installing minio with new version.go:"
	@godep go install github.com/minio/mc
	@mkdir -p $(HOME)/.mc

genversion:
	@echo "Generating a new version.go:"
	@godep go run genversion.go

coverage:
	@go test -race -coverprofile=cover.out
	@go tool cover -html=cover.out && echo "Visit your browser"

godepupdate:
	@(env bash $(PWD)/buildscripts/updatedeps.sh)
save:
	@godep save ./...

env:
	@godep go env

install: gomake-all

clean:
	@rm -fv cover.out
	@rm -fv mc
	@find Godeps -name "*.a" -type f -exec rm -vf {} \+
