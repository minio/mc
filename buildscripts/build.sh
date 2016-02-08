#!/bin/bash

# Save release LDFLAGS
LDFLAGS=$(go run buildscripts/gen-ldflags.go)

# Extract of release tag
release_tag=$(echo $LDFLAGS | awk {'print $4'} | cut -f2 -d=)

# Extract release string.
release_str=$(echo $MC_RELEASE | tr '[:upper:]' '[:lower:]')

# List of supported architectures
SUPPORTED_OSARCH='linux/386 linux/amd64 linux/arm windows/386 windows/amd64 darwin/amd64'

# Builds.
echo "Executing $release_str builds."
for osarch in ${SUPPORTED_OSARCH}; do
    os=$(echo $osarch | cut -f1 -d'/')
    arch=$(echo $osarch | cut -f2 -d'/')
    package=$(go list -f '{{.ImportPath}}')
    echo -n "-->"
    printf "%15s:%s\n" "${osarch}" "${package}"
    GOOS=$os GOARCH=$arch go build --ldflags "${LDFLAGS}" -o $release_str/$os-$arch/$(basename $package).$release_tag
done


