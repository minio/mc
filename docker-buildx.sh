#!/bin/bash
#
# Copyright (c) 2015-2023 MinIO, Inc.
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

sudo sysctl net.ipv6.conf.all.disable_ipv6=1

release=$(git describe --abbrev=0 --tags)

docker buildx build --push --no-cache \
	--build-arg RELEASE="${release}" \
	-t "minio/mc:latest" \
	-t "minio/mc:${release}" \
	-t "quay.io/minio/mc:${release}" \
	-t "quay.io/minio/mc:latest" \
	--platform=linux/arm64,linux/amd64,linux/ppc64le \
	-f Dockerfile.release .

docker buildx prune -f

docker buildx build --push --no-cache \
	--build-arg RELEASE="${release}" \
	-t "minio/mc:${release}-cpuv1" \
	-t "quay.io/minio/mc:${release}-cpuv1" \
	--platform=linux/arm64,linux/amd64,linux/ppc64le \
	-f Dockerfile.release.old_cpu .

docker buildx prune -f

sudo sysctl net.ipv6.conf.all.disable_ipv6=0
