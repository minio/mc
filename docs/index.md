# Minio Client

mc - Minio Client for S3 Compatible Object Storage released under [Apache license v2](./LICENSE).

## Install

```
# go get github.com/minio-io/mc
```

## Usage

### Commands
```
$ mc help

NAME:
   mc - Minio Client for S3 Compatible Object Storage

USAGE:
   mc [global options] command [command options] [arguments...]

VERSION:
   <---- GIT COMMIT ID ---->

BUILD:
   <---- BUILD TIME ---->

COMMANDS:
   cp	         copy objects and files
   ls	         list files and objects
   mb	         makes a bucket
   config        Generate configuration "/home/alexa/.mc/config.json" file.

GLOBAL OPTIONS:
   --debug       enable HTTP tracing
   --quiet, -q	 disable chatty output, such as the progress bar
   --version, -v print the version

```

## Contribute

[Contribute to mc](./contributing.md)

### Supported platforms

| Name  | Supported |
| ------------- | ------------- |
| Linux  | Yes  |
| Windows | Yes |
| Mac OSX | Yes |

### Supported architectures

| Arch | Supported |
| ------------- | ------------- |
| x86-64 | Yes |
| arm64 | Not yet|
| i386 | Not yet |

### Enable bash completion

To generate bash completion for ``mc`` all you have to do is

```
$ mc config --completion

Configuration written to /home/user/.mc/mc.bash_completion

$ source ${HOME}/.mc/mc.bash_completion
$ echo 'source ${HOME}/.mc/mc.bash_completion' >> ${HOME}/.bashrc

```

```
$ mc <TAB><TAB>
config  cp         h          help       ls         mb
```
