# Minio Client (mc) [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

``mc`` provides minimal tools to work with Amazon S3 compatible cloud storage and filesystems. It has features like resumable uploads, progress bar, parallel copy. ``mc`` is written in golang and released under [Apache license v2](./LICENSE).

## Commands

``mc`` implements the following commands
```
  ls		List files and folders.
  mb		Make a bucket or folder.
  cat		Display contents of a file.
  pipe		Write contents of stdin to one or more targets. When no target is specified, it writes to stdout.
  share		Generate URL for sharing.
  cp		Copy one or more objects to a target.
  mirror	Mirror folders recursively from a single source to many destinations.
  diff		Compute differences between two folders.
  rm		Remove file or bucket [WARNING: Use with care].
  access	Manage bucket access permissions.
  session	Manage saved sessions of cp and mirror operations.
  config	Manage configuration file.
  update	Check for a new software update.
  version	Print version.
```

## Install [![Build Status](https://api.travis-ci.org/minio/mc.svg?branch=master)](https://travis-ci.org/minio/mc) [![Build status](https://ci.appveyor.com/api/projects/status/3ng8bef7b3e1v763?svg=true)](https://ci.appveyor.com/project/harshavardhana/mc)

#### GNU/Linux

Download ``mc`` for:

- ``64-bit Intel`` from https://dl.minio.io:9000/updates/2015/Nov/linux-amd64/mc
- ``32-bit Intel`` from https://dl.minio.io:9000/updates/2015/Nov/linux-386/mc
- ``32-bit ARM`` from https://dl.minio.io:9000/updates/2015/Nov/linux-arm/mc

~~~
$ chmod +x mc
$ ./mc help
~~~

#### OS X

Download ``mc`` from https://dl.minio.io:9000/updates/2015/Nov/darwin-amd64/mc

~~~
$ chmod 755 mc
$ ./mc help
~~~

#### Microsoft Windows

Download ``mc`` for:

- ``64-bit`` from https://dl.minio.io:9000/updates/2015/Nov/windows-amd64/mc.exe
- ``32-bit`` from https://dl.minio.io:9000/updates/2015/Nov/windows-386/mc.exe

~~~
C:\Users\Username\Downloads> mc.exe help
~~~

#### Source
<blockquote>
NOTE:  Source installation is intended for only developers and advanced users. ‘mc update’ continous delivery mechanism is not supported for ‘go get’ based binary builds. Please download official releases from https://minio.io/#mc.
</blockquote>

If you do not have a working Golang environment, please follow [Install Golang](./INSTALLGO.md).

```sh
$ GO15VENDOREXPERIMENT=1 go get -u github.com/minio/mc
```

## Public Minio Server

Minio cloud storage server is hosted at ``https://play.minio.io:9000`` for public use. This service is primarily intended for developers and users to familiarize themselves with Amazon S3 compatible cloud storage. Minio runs with filesystem backend with auto-expiry for objects in about 24 hours.  No account signup is required, which means S3 compatible tools and applications can access this service without access and secret keys.

## Configuring mc for Amazon S3

Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSGettingStartedGuide/AWSCredentials.html).

Once you have them update your ``~/.mc/config.json`` configuration file.
```
$ mc config host add <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> S3v4
```

Example
```
$ mc config host add https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

NOTE: ``S3v4`` is default if not specified.

## Configure mc for Google Cloud Storage

Get your AccessKeyID and SecretAccessKey by following [Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys).

Once you have them update your ``~/./mc/config.json`` configuration file.
```
$ mc config host add https://storage.googleapis.com <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> S3v2
```

NOTE: Google Cloud Storage only supports Legacy Signature Version ``2``, so you have to pick - ``S3v2``

## Contribute to Minio Client
Please follow Minio [Contributor's Guide](./CONTRIBUTING.md)

### Jobs
If you think in Lisp or Haskell and hack in go, you would blend right in. Send your github link to callhome@minio.io.
