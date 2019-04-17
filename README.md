# MinIO Client Quickstart Guide
[![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io) [![Go Report Card](https://goreportcard.com/badge/minio/mc)](https://goreportcard.com/report/minio/mc) [![Docker Pulls](https://img.shields.io/docker/pulls/minio/mc.svg?maxAge=604800)](https://hub.docker.com/r/minio/mc/)

MinIO Client (mc) provides a modern alternative to UNIX commands like ls, cat, cp, mirror, diff, find etc. It supports filesystems and Amazon S3 compatible cloud storage service (AWS Signature v2 and v4).

```
ls       list buckets and objects
mb       make a bucket
rb       remove a bucket
cat      display object contents
head     display first 'n' lines of an object
pipe     stream STDIN to an object
share    generate URL for temporary access to an object
cp       copy objects
mirror   synchronize objects to a remote site
find     search for objects
sql      run sql queries on objects
stat     stat contents of objects
diff     list differences in object name, size, and date between buckets
rm       remove objects
event    manage object notifications
watch    watch for object events
policy   manage anonymous access to objects
admin    manage MinIO servers
session  manage saved sessions for cp command
config   manage mc configuration file
update   check for a new software update
version  print version info
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

**Note:** Above examples run `mc` against MinIO [_play_ environment](#test-your-setup) by default. To run `mc` against other S3 compatible servers, start the container this way:

```sh
docker run -it --entrypoint=/bin/sh minio/mc
```

then use the [`mc config` command](#add-a-cloud-storage-service).

## macOS
### Homebrew
Install mc packages using [Homebrew](http://brew.sh/)

```sh
brew install minio/stable/mc
mc --help
```

## GNU/Linux
### Binary Download
| Platform | Architecture | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.min.io/client/mc/release/linux-amd64/mc |
||64-bit PPC|https://dl.min.io/client/mc/release/linux-ppc64le/mc |

```sh
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc
./mc --help
```

## Microsoft Windows
### Binary Download
| Platform | Architecture | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit Intel|https://dl.min.io/client/mc/release/windows-amd64/mc.exe |

```sh
mc.exe --help
```

## Install from Source
Source installation is intended only for developers and advanced users. `mc update` command does not support update notifications for source based installations. Please download official releases from https://min.io/download/#minio-client.

If you do not have a working Golang environment, please follow [How to install Golang](https://docs.min.io/docs/how-to-install-golang).

```sh
go get -d github.com/minio/mc
cd ${GOPATH}/src/github.com/minio/mc
make
```

## Add a Cloud Storage Service
If you are planning to use `mc` only on POSIX compatible filesystems, you may skip this step and proceed to [everyday use](#everyday-use).

To add one or more Amazon S3 compatible hosts, please follow the instructions below. `mc` stores all its configuration information in ``~/.mc/config.json`` file.

```sh
mc config host add <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> --api <API-SIGNATURE> --lookup <BUCKET-LOOKUP-TYPE>
```

Alias is simply a short name to your cloud storage service. S3 end-point, access and secret keys are supplied by your cloud storage provider. API signature is an optional argument. By default, it is set to "S3v4".

Lookup is an optional argument. It is used to indicate whether dns or path style url requests are supported by the server. It accepts "dns", "path" or "auto" as valid values. By default, it is set to "auto" and SDK automatically determines the type of url lookup to use.

### Example - MinIO Cloud Storage
MinIO server displays URL, access and secret keys.

```sh
mc config host add minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

### Example - Amazon S3 Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/general/latest/gr/aws-security-credentials.html).

```sh
mc config host add s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

**Note**: As an IAM user on Amazon S3 you need to make sure the user has full access to the buckets or set the following restricted policy for your IAM user

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowBucketStat",
            "Effect": "Allow",
            "Action": [
                "s3:HeadBucket"
            ],
            "Resource": "*"
        },
        {
            "Sid": "AllowThisBucketOnly",
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": [
                "arn:aws:s3:::<your-restricted-bucket>/*",
                "arn:aws:s3:::<your-restricted-bucket>"
            ]
        }
    ]
}
```

### Example - Google Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys)

```sh
mc config host add gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

NOTE: Google Cloud Storage only supports Legacy Signature Version 2, so you have to pick - S3v2

## Test Your Setup
`mc` is pre-configured with https://play.min.io:9000, aliased as "play". It is a hosted MinIO server for testing and development purpose.  To test Amazon S3, simply replace "play" with "s3" or the alias you used at the time of setup.

*Example:*

List all buckets from https://play.min.io:9000

```sh
mc ls play
[2016-03-22 19:47:48 PDT]     0B my-bucketname/
[2016-03-22 22:01:07 PDT]     0B mytestbucket/
[2016-03-22 20:04:39 PDT]     0B mybucketname/
[2016-01-28 17:23:11 PST]     0B newbucket/
[2016-03-20 09:08:36 PDT]     0B s3git-test/
```

Make a bucket
`mb` command creates a new bucket.

*Example:*
```sh
mc mb play/mybucket
Bucket created successfully `play/mybucket`.
```

Copy Objects
`cp` command copies data from one or more sources to a target.

*Example:*
```sh
mc cp myobject.txt play/mybucket
myobject.txt:    14 B / 14 B  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  100.00 % 41 B/s 0
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
alias find='mc find'
```

### Shell autocompletion

In case you are using bash or zsh, the current shell will be automatically setup to use completion feature to suggest or auto-complete your mc commands.

```sh
mc <TAB>
admin    config   diff     find     ls       mirror   policy   session  sql      update   watch
cat      cp       event    head     mb       pipe     rm       share    stat     version
```

## Explore Further
- [MinIO Client Complete Guide](https://docs.min.io/docs/minio-client-complete-guide)
- [MinIO Quickstart Guide](https://docs.min.io/docs/minio-quickstart-guide)
- [The MinIO documentation website](https://docs.min.io)

## Contribute to MinIO Project
Please follow MinIO [Contributor's Guide](https://github.com/minio/mc/blob/master/CONTRIBUTING.md)


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fminio%2Fmc.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fminio%2Fmc?ref=badge_large)
