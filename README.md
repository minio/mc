# Minio Client

mc - Minio Client for S3 Compatible Object Storage released under [Apache license v2](./LICENSE).

## Install

```
# go get github.com/minio-io/mc
```

## Usage

### Commands
```
NAME:
   mc - Minio Client for S3 Compatible Object Storage

USAGE:
   mc [global options] command [command options] [arguments...]

VERSION:
   0.1.0

AUTHOR:
  Minio.io

COMMANDS:
   cp           copy objects
   ls           get list of objects
   mb           makes a bucket
   config       Generate configuration "/home/harsha/.minio/mc/config.json" file.
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug                      enable HTTP tracing
   --quiet, -q                  disable chatty output, such as the progress bar
   --get-bash-completion        Generate bash completion "/home/harsha/.minio/mc/mc.bash_completion" file.
   --help, -h                   show help
   --generate-bash-completion
   --version, -v                print the version
```

## Contribute

[Contribute to mc](./CONTRIBUTING.md)

### Enable bash completion

To generate bash completion for ``mc`` all you have to do is

```
$ mc --get-bash-completion

Configuration written to /home/user/.minio/mc/mc.bash_completion

$ source ${HOME}/.minio/mc/mc.bash_completion
$ echo 'source ${HOME}/.minio/mc/mc.bash_completion' >> ${HOME}/.bashrc

```

```
$ mc <TAB><TAB>
config  cp         h          help       ls         mb         sync
```
