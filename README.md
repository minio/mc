# Minio Client

mc - unified command line interface for Minio Object Storage compatible with Amazon S3 API released under [Apache license v2](./LICENSE).

## Install

```
# go get github.com/minio-io/mc
```

## Usage

### Commands
```
# mc --help
...
...
...
COMMANDS:
   s3
   s3api
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h           show help
   --version, -v        print the version
```

### Sub-Commands
```
# mc s3 --help
NAME:
   mc s3 -

USAGE:
   mc s3 command [command options] [arguments...]

COMMANDS:
   cp
   ls
   mkdir
   sync
   help, h      Shows a list of commands or help for one command

OPTIONS:
   --help, -h   show help

```

```
# mc s3api --help
NAME:
   mc s3api -

USAGE:
   mc s3api command [command options] [arguments...]

COMMANDS:
   get-object
   put-object
   put-bucket
   list-objects
   list-buckets
   configure
   help, h      Shows a list of commands or help for one command

OPTIONS:
   --help, -h   show help

```

## Contribute

[Contribute to mc](./CONTRIBUTING.md)
