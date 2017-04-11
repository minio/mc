# Minio Client Quickstart Guide [![Slack](https://slack.minio.io/slack?type=svg)](https://slack.minio.io)

Minio Client (mc) provides a modern alternative to UNIX commands like ls, cat, cp, mirror, diff etc. It supports filesystems and Amazon S3 compatible cloud storage service (AWS Signature v2 and v4).

```
ls            List files and folders.
mb            Make a bucket or folder.
cat           Display contents of a file.
pipe          Write contents of stdin to target. When no target is specified, it writes to stdout.
share         Generate URL for sharing.
cp            Copy one or more objects to a target.
mirror        Mirror folders recursively from a single source to single destination.
diff          Compute differences between two folders.
rm            Remove file or bucket [WARNING: Use with care].
events        Manage bucket notification.
watch         Watch for events on object storage and filesystem.
policy	      Set public policy on bucket or prefix.
session       Manage saved sessions of cp and mirror operations.
config        Manage configuration file.
update        Check for a new software update.
version       Print version.
```

## Docker Container
### Stable
```
docker pull minio/mc
docker run minio/mc ls play
```

### Edge
```
docker pull minio/mc:edge
docker run minio/mc:edge ls play
```

## macOS
### Homebrew
Install mc packages using [Homebrew](http://brew.sh/)

```sh
brew install minio-mc
mc --help
```

## GNU/Linux
### Binary Download
| Platform | Architecture | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.minio.io/client/mc/release/linux-amd64/mc |
|         |32-bit Intel|https://dl.minio.io/client/mc/release/linux-386/mc |
|         |32-bit ARM|https://dl.minio.io/client/mc/release/linux-arm/mc |

```sh
chmod +x mc
./mc --help
```

## Microsoft Windows
### Binary Download
| Platform | Architecture | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit|https://dl.minio.io/client/mc/release/windows-amd64/mc.exe |
|                 |32-bit|https://dl.minio.io/client/mc/release/windows-386/mc.exe  |

```sh
mc.exe --help
```

## FreeBSD
### Binary Download
| Platform | Architecture | URL |
| ---------- | -------- |------|
|FreeBSD|64-bit|https://dl.minio.io/client/mc/release/freebsd-amd64/mc |

```sh
chmod 755 mc
./mc --help
```

## Solaris/Illumos
### From Source

```sh
go get -u github.com/minio/mc
mc --help
```

## Install from Source
Source installation is intended only for developers and advanced users. `mc update` command does not support update notifications for source based installations. Please download official releases from https://minio.io/downloads/#minio-client.

If you do not have a working Golang environment, please follow [How to install Golang](https://docs.minio.io/docs/how-to-install-golang).

```sh
go get -u github.com/minio/mc
```

## Add a Cloud Storage Service
If you are planning to use `mc` only on POSIX compatible filesystems, you may skip this step and proceed to [everyday use](#everyday-use).

To add one or more Amazon S3 compatible hosts, please follow the instructions below. `mc` stores all its configuration information in ``~/.mc/config.json`` file.

```sh
mc config host add <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> <API-SIGNATURE>
```

Alias is simply a short name to you cloud storage service. S3 end-point, access and secret keys are supplied by your cloud storage provider. API signature is an optional argument. By default, it is set to "S3v4".

### Example - Minio Cloud Storage
Minio server displays URL, access and secret keys.

```sh
mc config host add minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v4
```

### Example - Amazon S3 Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/general/latest/gr/aws-security-credentials.html).

```sh
mc config host add s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v4
```

### Example - Google Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys)

```sh
mc config host add gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v2
```

NOTE: Google Cloud Storage only supports Legacy Signature Version 2, so you have to pick - S3v2

## Test Your Setup
`mc` is pre-configured with https://play.minio.io:9000, aliased as "play". It is a hosted Minio server for testing and development purpose.  To test Amazon S3, simply replace "play" with "s3" or the alias you used at the time of setup.

*Example:*

List all buckets from https://play.minio.io:9000

```sh
mc ls play
[2016-03-22 19:47:48 PDT]     0B my-bucketname/
[2016-03-22 22:01:07 PDT]     0B mytestbucket/
[2016-03-22 20:04:39 PDT]     0B mybucketname/
[2016-01-28 17:23:11 PST]     0B newbucket/
[2016-03-20 09:08:36 PDT]     0B s3git-test/
```
<a name="everyday-use"></a>
## Everyday Use

### Shell aliases
You may add shell aliases to override your common Unix tools.

```sh
alias ls='mc ls'
alias cp='mc cp'
alias cat='mc cat'
alias mkdir='mc mb'
alias pipe='mc pipe'
```

### Shell autocompletion
You may also download [`autocomplete/bash_autocomplete`](https://raw.githubusercontent.com/minio/mc/master/autocomplete/bash_complete) into `/etc/bash_completion.d/` and rename it to `mc`. Don't forget to source the file to make it active on your current shell.

```sh
sudo wget https://raw.githubusercontent.com/minio/mc/master/autocomplete/bash_complete -O /etc/bash_completion.d/mc
source /etc/bash_completion.d/mc
```

```sh
mc <TAB>
admin    config   diff     ls       mirror   policy   session  update   watch
cat      cp       events   mb       pipe     rm       share    version
```

## Explore Further
- [Minio Client Complete Guide](https://docs.minio.io/docs/minio-client-complete-guide)
- [Minio Quickstart Guide](https://docs.minio.io/docs/minio-quickstart-guide)
- [The Minio documentation website](https://docs.minio.io)

## Contribute to Minio Project
Please follow Minio [Contributor's Guide](https://github.com/minio/mc/blob/master/CONTRIBUTING.md)

