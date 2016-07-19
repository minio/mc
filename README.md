# Minio Client Quickstart Guide

Minio Client (mc) provides a modern alternative to UNIX commands like ls, cat, cp, mirror, diff etc. It supports filesystems and Amazon S3 compatible cloud storage service (AWS Signature v2 and v4).

```sh

ls            List files and folders.
mb            Make a bucket or folder.
cat           Display contents of a file.
pipe          Write contents of stdin to target. When no target is specified, it writes to stdout.
share         Generate URL for sharing.
cp            Copy one or more objects to a target.
mirror        Mirror folders recursively from a single source to single destination.
diff          Compute differences between two folders.
rm            Remove file or bucket [WARNING: Use with care].
policy        Set public policy on bucket or prefix.
session       Manage saved sessions of cp and mirror operations.
config        Manage configuration file.
update        Check for a new software update.
version       Print version.

```

## 1.  Download Minio Client

| Platform | Architecture | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.minio.io/client/mc/release/linux-amd64/mc|
||32-bit Intel|https://dl.minio.io/client/mc/release/linux-386/mc|
||32-bit ARM|https://dl.minio.io/client/mc/release/linux-arm/mc|
|Apple OS X|64-bit Intel|https://dl.minio.io/client/mc/release/darwin-amd64/mc|
|Microsoft Windows|64-bit|https://dl.minio.io/client/mc/release/windows-amd64/mc.exe|
||32-bit|https://dl.minio.io/server/minio/release/windows-386/minio.exe|
|FreeBSD|64-bit|https://dl.minio.io/client/mc/release/freebsd-amd64/mc|
|Solaris/Illumos|64-bit|https://dl.minio.io/client/mc/release/solaris-amd64/mc|

### Install from Source

Source installation is intended only for developers and advanced users. `mc update` command does not support upgrading from source based installation. Please download official releases from https://minio.io/downloads/#minio-client.

If you do not have a working Golang environment, please follow [How to install Golang](https://docs.minio.io/docs/how-to-install-golang).

```sh

$ go get -u github.com/minio/mc

```
## 2. Run Minio Client

### 1. GNU/Linux

```sh

$ chmod +x mc
$ ./mc --help

```

### 2. OS X

```sh

$ chmod 755 mc
$ ./mc --help

```

### 3. Microsoft Windows

```sh

C:\Users\Username\Downloads> mc.exe --help

```

### 4. Solaris/Illumos

```sh

$ chmod 755 mc
$ ./mc --help

```

### 5. FreeBSD

```sh

$ chmod 755 mc
$ ./mc --help

```

## 3. Add a Cloud Storage Service

Note: If you are planning to use `mc` only on POSIX compatible filesystems, you may skip this step and proceed to **Step 4**.

To add one or more Amazon S3 compatible hosts, please follow the instructions below. `mc` stores all its configuration information in ``~/.mc/config.json`` file.

#### Usage

```sh

mc config host add <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> <API-SIGNATURE>

```
Alias is simply a short name to you cloud storage service. S3 end-point, access and secret keys are supplied by your cloud storage provider. API signature is an optional argument. By default, it is set to "S3v4".

### Example - Minio Cloud Storage
Minio server displays URL, access and secret keys.

```sh

$ mc config host add minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v4

```
### Example - Amazon S3 Cloud Storage

Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSGettingStartedGuide/AWSCredentials.html).

```sh

$ mc config host add s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v4

```

### Example - Google Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys)

```sh

$ mc config host add gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v2

```

NOTE: Google Cloud Storage only supports Legacy Signature Version 2, so you have to pick - S3v2

## 4. Test Your Setup

`mc` is pre-configured with https://play.minio.io:9000, aliased as "play". It is a hosted Minio server for testing and development purpose.  To test Amazon S3, simply replace "play" with "s3" or the alias you used at the time of setup.

*Example:*

List all buckets from https://play.minio.io:9000

```sh

$ mc ls play
[2016-03-22 19:47:48 PDT]     0B my-bucketname/
[2016-03-22 22:01:07 PDT]     0B mytestbucket/
[2016-03-22 20:04:39 PDT]     0B mybucketname/
[2016-01-28 17:23:11 PST]     0B newbucket/
[2016-03-20 09:08:36 PDT]     0B s3git-test/

```
## 5. Everyday Use

You may add shell aliases to override your common Unix tools.

```sh

alias ls='mc ls'
alias cp='mc cp'
alias cat='mc cat'
alias mkdir='mc mb'
alias pipe='mc pipe'

```
## 6. Explore Further

- [Minio Client Complete Guide](https://docs.minio.io/docs/minio-client-complete-guide)
- [Minio Quickstart Guide](https://docs.minio.io/docs/minio)

## 7. Contribute
[Contributors Guide](./CONTRIBUTING.md)
