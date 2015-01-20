package main

import "errors"

// api
var objectBlobErr = errors.New("object blob is mandatory")
var objectNameErr = errors.New("object name is mandatory")
var bucketNameErr = errors.New("bucket name is mandatory")

// fs
var fsPathErr = errors.New("Arguments missing <S3Path> or <LocalPath>")
var fsUriErr = errors.New("Invalid URI scheme")

// configure
var configAccessErr = errors.New("accesskey is mandatory")
var configSecretErr = errors.New("secretkey is mandatory")

// common
var missingAccessSecretErr = errors.New("You can configure your credentials by running `mc configure`")
var missingAccessErr = errors.New("Partial credentials found in the env, missing : AWS_ACCESS_KEY_ID")
var missingSecretErr = errors.New("Partial credentials found in the env, missing : AWS_SECRET_ACCESS_KEY")
