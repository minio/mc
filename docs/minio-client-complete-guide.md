# MinIO Client Complete Guide [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io)

MinIO Client (mc) provides a modern alternative to UNIX commands like ls, cat, cp, mirror, diff etc. It supports filesystems and Amazon S3 compatible cloud storage service (AWS Signature v2 and v4).

```
alias      manage server credentials in configuration file
ls         list buckets and objects
mb         make a bucket
rb         remove a bucket
cp         copy objects
mv         move objects
rm         remove object(s)
mirror     synchronize object(s) to a remote site
cat        display object contents
head       display first 'n' lines of an object
pipe       stream STDIN to an object
find       search for objects
sql        run sql queries on objects
stat       show object metadata
tree       list buckets and objects in a tree format
du         summarize disk usage recursively
retention  set retention for object(s)
legalhold  manage legal hold for object(s)
support    support related commands
license    license related commands
share      generate URL for temporary access to an object
version    manage bucket versioning
ilm        manage bucket lifecycle
quota      manage bucket quota
encrypt    manage bucket encryption config
event      manage object notifications
watch      listen for object notification events
undo       undo PUT/DELETE operations
anonymous  manage anonymous access to buckets and objects
tag        manage tags for bucket and object(s)
diff       list differences in object name, size, and date between two buckets
replicate  configure server side bucket replication
admin      manage MinIO servers
update     update mc to latest release
ready      checks if the cluster is ready or not
ping       perform liveness check
od         measure single stream upload and download
batch      manage batch jobs
```

## 1.  Download MinIO Client
### Docker Stable
```
docker pull minio/mc
docker run minio/mc ls play
```

### Docker Edge
```
docker pull minio/mc:edge
docker run minio/mc:edge ls play
```

**Note:** Above examples run `mc` against MinIO [_play_ environment](#test-your-setup) by default. To run `mc` against other S3 compatible servers, start the container this way:

```
docker run -it --entrypoint=/bin/sh minio/mc
```

then use the [`mc alias` command](#3-add-a-cloud-storage-service).

### Homebrew (macOS)
Install mc packages using [Homebrew](http://brew.sh/)

```
brew install minio/stable/mc
mc --help
```

### Binary Download (GNU/Linux)
| Platform | Architecture | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.min.io/client/mc/release/linux-amd64/mc |
||64-bit PPC|https://dl.min.io/client/mc/release/linux-ppc64le/mc |

```
chmod +x mc
./mc --help
```

### Binary Download (Microsoft Windows)
| Platform | Architecture | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit Intel|https://dl.min.io/client/mc/release/windows-amd64/mc.exe |

```
mc.exe --help
```

### Install from Source
Source installation is intended only for developers and advanced users. `mc update` command does not support update notifications for source based installations. Please download official releases from https://min.io/download/#minio-client.

If you do not have a working Golang environment, please follow [How to install Golang](https://golang.org/doc/install).

```
go get -d github.com/minio/mc
cd ${GOPATH}/src/github.com/minio/mc
make
```

## 2. Run MinIO Client

### GNU/Linux

```
chmod +x mc
./mc --help
```

### macOS

```
chmod 755 mc
./mc --help
```

### Microsoft Windows

```
mc.exe --help
```

## 3. Add a Cloud Storage Service
Note: If you are planning to use `mc` only on POSIX compatible filesystems, you may skip this step and proceed to **Step 4**.

To add one or more Amazon S3 compatible hosts, please follow the instructions below. `mc` stores all its configuration information in ``~/.mc/config.json`` file.

#### Usage

```
mc alias set <ALIAS> <YOUR-S3-ENDPOINT> [YOUR-ACCESS-KEY] [YOUR-SECRET-KEY] [--api API-SIGNATURE]
```

Keys must be supplied by argument or standard input.

Alias is simply a short name to your cloud storage service. S3 end-point, access and secret keys are supplied by your cloud storage provider. API signature is an optional argument. By default, it is set to "S3v4".

### Example - MinIO Cloud Storage
MinIO server displays URL, access and secret keys.


```
mc alias set minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 --api S3v4
```

### Example - Amazon S3 Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [AWS Credentials Guide](http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSGettingStartedGuide/AWSCredentials.html).

```
mc alias set s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 --api S3v4
```

### Example - Google Cloud Storage
Get your AccessKeyID and SecretAccessKey by following [Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys)

```
mc alias set gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

### Example - IBM Cloud Object Storage
Get your AccessKeyID and SecretAccessKey by creating a service account [with HMAC credentials](https://cloud.ibm.com/docs/cloud-object-storage?topic=cloud-object-storage-uhc-hmac-credentials-main).  This option is only available from **Resources > Cloud Object Storage > Service credentials** (not from Manage > Access (IAM) > Service IDs). Once created, the values you will use for `accessKey` and `secretKey` are found in the `cos_hmac_keys` field of the service credentials.

Finally, the url will be the **public endpoint specific to the region/resiliency** that you chose when setting up your bucket. There is no single, global url for all buckets. Find your bucket's URL in the console by going to Cloud Object Storage > Buckets > [your-bucket] > Configuration > Endpoints > public. Remember to prepend `https://` to the URL provided.

```
mc alias set ibm https://s3.us-east.cloud-object-storage.appdomain.cloud BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 --api s3v4
```

**Note**: The service ID you create must have an access policy granting it access to your Object Storage instance(s). 

### Example - Specify keys using standard input

#### Prompt

```
mc alias set minio http://192.168.1.51 --api S3v4
Enter Access Key: BKIKJAA5BMMU2RHO6IBB
Enter Secret Key: V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

#### Pipe from STDIN

```
echo -e "BKIKJAA5BMMU2RHO6IBB\nV7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12" | \
mc alias set minio http://192.168.1.51 --api S3v4
```

### Specify temporary host configuration through environment variable

#### Static credentials
```
export MC_HOST_<alias>=https://<Access Key>:<Secret Key>@<YOUR-S3-ENDPOINT>
```

Example:
```
export MC_HOST_myalias=https://Q3AM3UQ867SPQQA43P2F:zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG@play.min.io
mc ls myalias
```

#### Rotating credentials
```
export MC_HOST_<alias>=https://<Access Key>:<Secret Key>:<Session Token>@<YOUR-S3-ENDPOINT>
```

Example:
```
export MC_HOST_myalias=https://Q3AM3UQ867SPQQA43P2F:zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG:eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiJOVUlCT1JaWVRWMkhHMkJNUlNYUiIsImF1ZCI6IlBvRWdYUDZ1Vk80NUlzRU5SbmdEWGo1QXU1WWEiLCJhenAiOiJQb0VnWFA2dVZPNDVJc0VOUm5nRFhqNUF1NVlhIiwiZXhwIjoxNTM0ODk2NjI5LCJpYXQiOjE1MzQ4OTMwMjksImlzcyI6Imh0dHBzOi8vbG9jYWxob3N0Ojk0NDMvb2F1dGgyL3Rva2VuIiwianRpIjoiNjY2OTZjZTctN2U1Ny00ZjU5LWI0MWQtM2E1YTMzZGZiNjA4In0.eJONnVaSVHypiXKEARSMnSKgr-2mlC2Sr4fEGJitLcJF_at3LeNdTHv0_oHsv6ZZA3zueVGgFlVXMlREgr9LXA@play.min.io
mc ls myalias
```


## 4. Test Your Setup
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

## 5. Everyday Use
You may add shell aliases to override your common Unix tools.

```
alias ls='mc ls'
alias cp='mc cp'
alias cat='mc cat'
alias mkdir='mc mb'
alias pipe='mc pipe'
alias find='mc find'
alias tree='mc tree'
```

## 6. Global Options

### Option [--autocompletion]
Install auto-completion for your shell.

### Option [--debug]
Debug option enables debug output to console.

*Example: Display verbose debug output for `ls` command.*

```
mc --debug ls play
mc: <DEBUG> GET / HTTP/1.1
Host: play.min.io
User-Agent: MinIO (darwin; amd64) minio-go/1.0.1 mc/2016-04-01T00:22:11Z
Authorization: AWS4-HMAC-SHA256 Credential=**REDACTED**/20160408/us-east-1/s3/aws4_request, SignedHeaders=expect;host;x-amz-content-sha256;x-amz-date, Signature=**REDACTED**
Expect: 100-continue
X-Amz-Content-Sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
X-Amz-Date: 20160408T145236Z
Accept-Encoding: gzip

mc: <DEBUG> HTTP/1.1 200 OK
Transfer-Encoding: chunked
Accept-Ranges: bytes
Content-Type: text/xml; charset=utf-8
Date: Fri, 08 Apr 2016 14:54:55 GMT
Server: MinIO/DEVELOPMENT.2016-04-07T18-53-27Z (linux; amd64)
Vary: Origin
X-Amz-Request-Id: HP30I0W2U49BDBIO

mc: <DEBUG> Response Time:  1.220112837s

[...]

[2016-04-08 03:56:14 IST]     0B albums/
[2016-04-04 16:11:45 IST]     0B backup/
[2016-04-01 20:10:53 IST]     0B deebucket/
[2016-03-28 21:53:49 IST]     0B guestbucket/
```

### Option [--json]
JSON option enables parseable output in [JSON lines](http://jsonlines.org/), also called as [NDJSON](http://ndjson.org/) format.

*Example: List all buckets from MinIO play service.*

```
mc --json ls play
{"status":"success","type":"folder","lastModified":"2016-04-08T03:56:14.577+05:30","size":0,"key":"albums/"}
{"status":"success","type":"folder","lastModified":"2016-04-04T16:11:45.349+05:30","size":0,"key":"backup/"}
{"status":"success","type":"folder","lastModified":"2016-04-01T20:10:53.941+05:30","size":0,"key":"deebucket/"}
{"status":"success","type":"folder","lastModified":"2016-03-28T21:53:49.217+05:30","size":0,"key":"guestbucket/"}
```

### Option [--no-color]
This option disables the color theme. It is useful for dumb terminals.

### Option [--quiet]
Quiet option suppress chatty console output.

### Option [--config-dir]
Use this option to set a custom config path.

### Option [ --insecure]
Skip SSL certificate verification.

### Option [--version]
Display the current version of `mc` installed

*Example: Print version of mc.*

```
mc --version
mc version RELEASE.2020-04-25T00-43-23Z
```

## 7. Commands

|                                                                                                                                                   |                                                                                                                                                                                |                                                                                                                                                              |                                                    |
|:--------------------------------------------------------------------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------|
| [**ls** - list buckets and objects](https://min.io/docs/minio/linux/reference/minio-mc/mc-ls.html)                                                | [**tree** - list buckets and objects in a tree format](https://min.io/docs/minio/linux/reference/minio-mc/mc-tree.html)                                                        | [**mb** - make a bucket](https://min.io/docs/minio/linux/reference/minio-mc/mc-mb.html)                                                                      | [**cat** - display object contents](https://min.io/docs/minio/linux/reference/minio-mc/mc-cat.html)          |
| [**cp** - copy objects](https://min.io/docs/minio/linux/reference/minio-mc/mc-cp.html)                                                            | [**rb** - remove a bucket](https://min.io/docs/minio/linux/reference/minio-mc/mc-rb.html)                                                                                      | [**pipe** - stream STDIN to an object](https://min.io/docs/minio/linux/reference/minio-mc/mc-pipe.html)                                                      | [**version** - manage bucket version](https://min.io/docs/minio/linux/reference/minio-mc/mc-version.html)    |
| [**share** - generate URL for temporary access to an object](https://min.io/docs/minio/linux/reference/minio-mc/mc-share.html)                    | [**rm** - remove objects](https://min.io/docs/minio/linux/reference/minio-mc/mc-rm.html)                                                                                       | [**find** - find files and objects](https://min.io/docs/minio/linux/reference/minio-mc/mc-find.html)                                                         | [**undo** - undo PUT/DELETE operations](https://min.io/docs/minio/linux/reference/minio-mc/mc-undo.html)     |
| [**diff** - list differences in object name, size, and date between two buckets](https://min.io/docs/minio/linux/reference/minio-mc/mc-diff.html) | [**mirror** - synchronize object(s) to a remote site](https://min.io/docs/minio/linux/reference/minio-mc/mc-mirror.html)                                                       | [**ilm** - manage bucket lifecycle policies](https://min.io/docs/minio/linux/reference/minio-mc/mc-ilm.html)                                                 | [**replicate** - manage bucket server side replication](https://min.io/docs/minio/linux/reference/minio-mc/mc-replicate.html) |
| [**alias** - manage aliases](https://min.io/docs/minio/linux/reference/minio-mc/mc-alias.html)                                                    | [**anonymous** - manage anonymous access to buckets and objects](#https://min.io/docs/minio/linux/reference/minio-mc/mc-anonymous.html)                                        | [**event** - manage events on your buckets](https://min.io/docs/minio/linux/reference/minio-mc/mc-event.html)                                                | [**encrypt** - manage bucket encryption](https://min.io/docs/minio/linux/reference/minio-mc/mc-encrypt.html) |
| [**update** - manage software updates](https://min.io/docs/minio/linux/reference/minio-mc/mc-update.html)                                         | [**watch** - watch for events](https://min.io/docs/minio/linux/reference/minio-mc/mc-watch.html)                                                                               | [**retention** - set retention for object(s)](https://min.io/docs/minio/linux/reference/minio-mc/mc-retention.html)                                          | [**sql** - run sql queries on objects](https://min.io/docs/minio/linux/reference/minio-mc/mc-sql.html)       |
| [**head** - display first 'n' lines of an object](https://min.io/docs/minio/linux/reference/minio-mc/mc-head.html)                                | [**stat** - stat contents of objects and folders](https://min.io/docs/minio/linux/reference/minio-mc/mc-stat.html)                                                             | [**legalhold** - set legal hold for object(s)](https://min.io/docs/minio/linux/reference/minio-mc/mc-legalhold.html)                                         | [**mv** - move objects](https://min.io/docs/minio/linux/reference/minio-mc/mc-mv.html)                       |
| [**du** - summarize disk usage recursively](https://min.io/docs/minio/linux/reference/minio-mc/mc-du.html)                                        | [**tag** - manage tags for bucket and object(s)](https://min.io/docs/minio/linux/reference/minio-mc/mc-tag.html)                                                               | [**admin** - manage MinIO servers](https://min.io/docs/minio/linux/reference/minio-mc-admin.html)                                                            | [**support** - generate profile data for debugging purposes](https://min.io/docs/minio/linux/reference/minio-mc/mc-support.html) |
| [**ping** - perform liveness check](https://min.io/docs/minio/linux/reference/minio-mc/mc-ping.html)                                              | [**license** - license related commands](https://min.io/docs/minio/linux/reference/minio-mc/mc-license.html)                                                                   | [**quota** - manage bucket quota](https://min.io/docs/minio/linux/reference/minio-mc/mc-quota.html)                                                          | [**watch** - listen for object notification events](https://min.io/docs/minio/linux/reference/minio-mc/mc-watch.html)                                                 |
| [**od** - measure single stream upload and download](https://min.io/docs/minio/linux/reference/minio-mc/mc-od.html)                               | [**batch** - manage batch jobs](https://min.io/docs/minio/linux/reference/minio-mc/mc-batch.html)                                                                              |                                                                                                                                                              |     |
