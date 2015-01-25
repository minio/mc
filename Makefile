all: getdeps install

checkdeps:
	@./checkdeps.sh

getdeps: checkdeps
	@go get github.com/tools/godep && echo "Installed godep"
	@go get golang.org/x/tools/cmd/cover && echo "Installed cover"

install: pkgs uri
	@godep go install github.com/minio-io/mc && echo "Installed mc"

pkgs:
	@godep go test -race -coverprofile=cover.out github.com/minio-io/mc/pkg/s3
	@godep go test -race -coverprofile=cover.out github.com/minio-io/mc/pkg/minio

uri:
	@godep go test -race -coverprofile=cover.out github.com/minio-io/mc/pkg/uri
