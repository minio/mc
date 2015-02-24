/*
 * Mini Object Storage, (C) 2014,2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import "errors"

// api
var objectBlobErr = errors.New("object blob is mandatory")
var objectNameErr = errors.New("object name is mandatory")
var bucketNameErr = errors.New("bucket name is mandatory")

// fs
var fsPathErr = errors.New("Arguments missing <S3Path> or <LocalPath>")
var fsUriErr = errors.New("Invalid URI scheme")
var fsKeyErr = errors.New("Key is needed to get the file")

// configure
var configAccessErr = errors.New("accesskey is mandatory")
var configSecretErr = errors.New("secretkey is mandatory")
var configEndpointErr = errors.New("endpoint is missing")

// common
var missingAccessSecretErr = errors.New("You can configure your credentials by running `mc configure`")
var missingAccessErr = errors.New("Partial credentials found in the env, missing : AWS_ACCESS_KEY_ID")
var missingSecretErr = errors.New("Partial credentials found in the env, missing : AWS_SECRET_ACCESS_KEY")
var invalidBucketErr = errors.New("Invalid bucket name")
