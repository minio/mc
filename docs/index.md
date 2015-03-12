# Minio Client

mc - unified command line interface for Minio and S3 released under [Apache license v2](./LICENSE).

## Install

```
# go get github.com/minio-io/mc
```

## Usage

### Commands
```
NAME:
   mc - unified command line interface for Minio and S3

USAGE:
   mc [global options] command [command options] [arguments...]

VERSION:
   0.1.0

AUTHOR:
  Minio Community

COMMANDS:
   cp
   ls
   mb
   sync
   configure
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h           show help
   --version, -v        print the version
```

## Contribute

[Contribute to mc](./contributing.md)

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
configure  cp         h          help       ls         mb         sync
```
