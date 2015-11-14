# Minio Client (mc) [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

``mc`` provides minimal tools to work with Amazon S3 compatible cloud storage and filesystems. It has features like resumable partial uploads, progress bar, parallel copy. ``mc`` is written in golang and released under [Apache license v2](./LICENSE).

## Commands

``mc`` implements the following commands
```
  ls        List files and folders.
  mb		Make a bucket or folder.
  cat		Display contents of a file.
  rm		Remove file or bucket.
  pipe	    Write contents of stdin to one or more targets. Pipe is the opposite of cat command.
  cp		Copy files and folders from many sources to a single destination.
  mirror	Mirror folders recursively from a single source to many destinations.
  session	Manage sessions for cp and mirror.
  share		Share documents via URL.
  diff		Compute differences between two files or folders.
  access	Set or get access permissions.
  config	Collection of config management commands.
  update	Check for new software updates.
  version	Print version.
```

## Install [![Build Status](https://api.travis-ci.org/minio/mc.svg?branch=master)](https://travis-ci.org/minio/mc) [![Build status](https://ci.appveyor.com/api/projects/status/3ng8bef7b3e1v763?svg=true)](https://ci.appveyor.com/project/harshavardhana/mc)

#### GNU/Linux

##### 64bit

Download ``mc`` from https://dl.minio.io:9000/updates/2015/Nov/linux-amd64/mc

##### 32bit 

Download ``mc`` from https://dl.minio.io:9000/updates/2015/Nov/linux-386/mc

##### Arm32

Download ``mc`` from https://dl.minio.io:9000/updates/2015/Nov/linux-arm/mc

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

#### Windows 64bit and 32bit

##### 64 bit

Download ``mc`` from https://dl.minio.io:9000/updates/2015/Nov/windows-amd64/mc.exe

##### 32 bit 

Download ``mc`` from https://dl.minio.io:9000/updates/2015/Nov/windows-386/mc.exe

~~~
C:\Users\Username\Downloads> mc.exe help
~~~

#### Source

If you do not have a working Golang environment, please follow [Install Golang](./INSTALLGO.md).

```sh
$ go get -u github.com/minio/mc
```

## Public Minio Server

Minio server is hosted at ``https://play.minio.io:9000`` for public use. This service is primarily intended for developers and users to familiarize themselves with Amazon S3 compatible cloud storage. Minio runs in memory mode with auto expiry of objects in about an hour.  No account signup is required, which means S3 compatible tools and applications can access this service without access and secret keys.

## How to use mc?

[![asciicast](https://asciinema.org/a/21576.png)](https://asciinema.org/a/21576?async)

## Configuring mc for Amazon S3

Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSGettingStartedGuide/AWSCredentials.html). 

Once you have them update your ``~/.mc/config.json`` configuration file.
```
$ mc config host add <your_s3_endpoint> <your_access_key> <your_secret_key> S3v4
```

NOTE: ``S3v4`` is default unless specified. 

## Configure mc for Google Cloud Storage

Get your AccessKeyID and SecretAccessKey by following [Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys). 

Once you have them update your ``~/./mc/config.json`` configuration file.
```
$ mc config host add storage.googleapis.com <your_access_key> <your_secret_key> S3v2
```

NOTE: Google Cloud Storage only supports Legacy Signature Version ``2``, so you have to pick - ``S3v2``

## Contribute to Minio Client
Please follow Minio [Contributor's Guide](./CONTRIBUTING.md)

### Jobs
If you think in Lisp or Haskell and hack in go, you would blend right in. Send your github link to callhome@minio.io.
