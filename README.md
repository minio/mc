# Minio Client (mc) [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

``mc`` provides minimal tools to work with Amazon S3 compatible cloud storage and filesystems. It has features like resumable uploads/downloads, progress bar, mirroring etc. ``mc`` is written in golang and released under [Apache license v2](./LICENSE).

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

- ``64-bit Intel`` from https://dl.minio.io/client/mc/release/linux-amd64/mc
- ``32-bit Intel`` from https://dl.minio.io/client/mc/release/linux-386/mc
- ``32-bit ARM`` from https://dl.minio.io/client/mc/release/linux-arm/mc

~~~
$ chmod 755 mc
$ ./mc --help
~~~

#### OS X

Download ``mc`` from https://dl.minio.io/client/mc/release/darwin-amd64/mc

~~~
$ chmod 755 mc
$ ./mc --help
~~~

#### Microsoft Windows

Download ``mc`` for:

- ``64-bit`` from https://dl.minio.io/client/mc/release/windows-amd64/mc.exe
- ``32-bit`` from https://dl.minio.io/client/mc/release/windows-386/mc.exe

Extract the downloaded zip file.

~~~
C:\Users\Username\Downloads> mc.exe --help
~~~

#### Solaris/Illumos

Download ``mc`` from https://dl.minio.io/client/mc/release/solaris-amd64/mc

~~~
$ chmod 755 mc
$ ./mc --help
~~~

#### FreeBSD

Download ``mc`` from https://dl.minio.io/client/mc/release/freebsd-amd64/mc

~~~
$ chmod 755 mc
$ ./mc --help
~~~

#### Source
<blockquote>
NOTE:  Source installation is intended for only developers and advanced users. ‘mc update’ continuous delivery mechanism is not supported for ‘go get’ based binary builds. Please download official releases from https://minio.io/#mc.
</blockquote>

If you do not have a working Golang environment, please follow [Install Golang](./INSTALLGO.md).

```sh
$ GO15VENDOREXPERIMENT=1 go get -u github.com/minio/mc
```

## Minio Test Server
Minio test server is hosted at ``https://play.minio.io:9000`` for public use. `mc` is pre-configured to use this service as 'play' alias. Access and secret keys for this server is saved in your `~/.mc/config.json`.
```
$ ./mc mb play/myownbucket
$ ./mc ls play
```

## Configuring mc for Amazon S3

Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSGettingStartedGuide/AWSCredentials.html).

Once you have them update your ``~/.mc/config.json`` configuration file.
```
$ ./mc config host add <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> S3v4
```

Example
```
$ ./mc config host add mys3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

NOTE: ``S3v4`` is default if not specified.

## Configure mc for Google Cloud Storage

Get your AccessKeyID and SecretAccessKey by following [Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys).

Once you have them update your ``~/.mc/config.json`` configuration file.
```
$ ./mc config host add <ALIAS> https://storage.googleapis.com <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> S3v2
```

NOTE: Google Cloud Storage only supports Legacy Signature Version ``2``, so you have to pick - ``S3v2``

## Contribute to Minio Client
Please follow Minio [Contributor's Guide](./CONTRIBUTING.md)
