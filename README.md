# MinIO Client Quickstart Guide
[![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io) [![Go Report Card](https://goreportcard.com/badge/minio/mc)](https://goreportcard.com/report/minio/mc) [![Docker Pulls](https://img.shields.io/docker/pulls/minio/mc.svg?maxAge=604800)](https://hub.docker.com/r/minio/mc/) [![license](https://img.shields.io/badge/license-AGPL%20V3-blue)](https://github.com/minio/mc/blob/master/LICENSE)

# Documentation
- [MC documentation](https://docs.min.io/community/minio-object-store/reference/minio-mc.html)

MinIO Client (mc) provides a modern alternative to UNIX commands like ls, cat, cp, mirror, diff, find etc. It supports filesystems and Amazon S3 compatible cloud storage service (AWS Signature v2 and v4).

```
  alias      manage server credentials in configuration file
  admin      manage MinIO servers
  anonymous  manage anonymous access to buckets and objects
  batch      manage batch jobs
  cp         copy objects
  cat        display object contents
  diff       list differences in object name, size, and date between two buckets
  du         summarize disk usage recursively
  encrypt    manage bucket encryption config
  event      manage object notifications
  find       search for objects
  get        get s3 object to local
  head       display first 'n' lines of an object
  ilm        manage bucket lifecycle
  idp        manage MinIO IDentity Provider server configuration
  license    license related commands
  legalhold  manage legal hold for object(s)
  ls         list buckets and objects
  mb         make a bucket
  mv         move objects
  mirror     synchronize object(s) to a remote site
  od         measure single stream upload and download
  ping       perform liveness check
  pipe       stream STDIN to an object
  put        upload an object to a bucket
  quota      manage bucket quota
  rm         remove object(s)
  retention  set retention for object(s)
  rb         remove a bucket
  replicate  configure server side bucket replication
  ready      checks if the cluster is ready or not
  sql        run sql queries on objects
  stat       show object metadata
  support    support related commands
  share      generate URL for temporary access to an object
  tree       list buckets and objects in a tree format
  tag        manage tags for bucket and object(s)
  undo       undo PUT/DELETE operations
  update     update mc to latest release
  version    manage bucket versioning
  watch      listen for object notification events
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

```
docker run -it --entrypoint=/bin/sh minio/mc
```

then use the [`mc alias` command](#add-a-cloud-storage-service).

### GitLab CI
When using the Docker container in GitLab CI, you must [set the entrypoint to an empty string](https://docs.gitlab.com/ee/ci/docker/using_docker_images.html#override-the-entrypoint-of-an-image).

```
deploy:
  image:
    name: minio/mc
    entrypoint: ['']
  stage: deploy
  before_script:
    - mc alias set minio $MINIO_HOST $MINIO_ACCESS_KEY $MINIO_SECRET_KEY
  script:
    - mc cp <source> <destination>
```

## macOS
### Homebrew
Install mc packages using [Homebrew](http://brew.sh/)

```
brew install minio/stable/mc
mc --help
```

## GNU/Linux
### Binary Download
| Platform | Architecture | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.min.io/client/mc/release/linux-amd64/mc |
|GNU/Linux|64-bit PPC|https://dl.min.io/client/mc/release/linux-ppc64le/mc |
|GNU/Linux|64-bit ARM|https://dl.min.io/client/mc/release/linux-arm64/mc |
|Linux/s390x|S390X|https://dl.min.io/client/mc/release/linux-s390x/mc |

```
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc
./mc --help
```

## Microsoft Windows
### Binary Download
| Platform | Architecture | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit Intel|https://dl.min.io/client/mc/release/windows-amd64/mc.exe |

```
mc.exe --help
```

## Install from Source
Source installation is only intended for developers and advanced users. If you do not have a working Golang environment, please follow [How to install Golang](https://golang.org/doc/install). Minimum version required is [go1.22](https://golang.org/dl/#stable)

```sh
go install github.com/minio/mc@latest
```

## Add a Cloud Storage Service
If you are planning to use `mc` only on POSIX compatible filesystems, you may skip this step and proceed to [everyday use](#everyday-use).

To add one or more Amazon S3 compatible hosts, please follow the instructions below. `mc` stores all its configuration information in ``~/.mc/config.json`` file.

```
mc alias set <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> --api <API-SIGNATURE> --path <BUCKET-LOOKUP-TYPE>
```

`<ALIAS>` is simply a short name to your cloud storage service. S3 end-point, access and secret keys are supplied by your cloud storage provider. API signature is an optional argument. By default, it is set to "S3v4".

Path is an optional argument. It is used to indicate whether dns or path style url requests are supported by the server. It accepts "on", "off" as valid values to enable/disable path style requests.. By default, it is set to "auto" and SDK automatically determines the type of url lookup to use.

### Example - MinIO Cloud Storage
MinIO server startup banner displays URL, access and secret keys.

```
mc alias set minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

### Example - Amazon S3 Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/general/latest/gr/aws-security-credentials.html).

```
mc alias set s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
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

```
mc alias set gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

## Test Your Setup
`mc` is pre-configured with https://play.min.io, aliased as "play". It is a hosted MinIO server for testing and development purpose.  To test Amazon S3, simply replace "play" with "s3" or the alias you used at the time of setup.

*Example:*

List all buckets from https://play.min.io

```
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
```
mc mb play/mybucket
Bucket created successfully `play/mybucket`.
```

Copy Objects
`cp` command copies data from one or more sources to a target.

*Example:*
```
mc cp myobject.txt play/mybucket
myobject.txt:    14 B / 14 B  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  100.00 % 41 B/s 0
```

## Everyday Use

### Shell aliases
You may add shell aliases to override your common Unix tools.

```
alias ls='mc ls'
alias cp='mc cp'
alias cat='mc cat'
alias mkdir='mc mb'
alias pipe='mc pipe'
alias find='mc find'
```

### Shell autocompletion
In case you are using bash, zsh or fish. Shell completion is embedded by default in `mc`, to install auto-completion use `mc --autocompletion`. Restart the shell, mc will auto-complete commands as shown below.

```
mc <TAB>
admin    config   diff     find     ls       mirror   policy   session  sql      update   watch
cat      cp       event    head     mb       pipe     rm       share    stat     version
```

## Contribute to MinIO Project
Please follow MinIO [Contributor's Guide](https://github.com/minio/mc/blob/master/CONTRIBUTING.md)

## License
Use of `mc` is governed by the GNU AGPLv3 license that can be found in the [LICENSE](https://github.com/minio/mc/blob/master/LICENSE) file.
