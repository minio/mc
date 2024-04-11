#!/bin/bash
#
# Copyright (c) 2015-2021 MinIO, Inc.
#
# This file is part of MinIO Object Storage stack
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#

set -e

# Enable tracing if set.
[ -n "$BASH_XTRACEFD" ] && set -x

function _init() {
    ## All binaries are static make sure to disable CGO.
    export CGO_ENABLED=0

    ## List of architectures and OS to test coss compilation.
    SUPPORTED_OSARCH="linux/ppc64le linux/s390x linux/mips64 linux/amd64 windows/amd64 darwin/amd64 darwin/arm64 linux/arm64 linux/arm"
}

function _build() {
    local osarch=$1
    IFS=/ read -r -a arr <<<"$osarch"
    os="${arr[0]}"
    arch="${arr[1]}"
    package=$(go list -f '{{.ImportPath}}')
    printf -- "--> %15s:%s\n" "${osarch}" "${package}"

    # Go build to build the binary.
    export GOOS=$os
    export GOARCH=$arch
    export GO111MODULE=on
    export CGO_ENABLED=0
    go build -tags kqueue -o /dev/null
}

function main() {
    echo "Testing builds for OS/Arch: ${SUPPORTED_OSARCH}"
    for each_osarch in ${SUPPORTED_OSARCH}; do
        _build "${each_osarch}"
    done
}

_init && main "$@"
