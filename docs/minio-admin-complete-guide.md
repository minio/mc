# MinIO Admin Complete Guide [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io)

MinIO Client (mc) provides `admin` sub-command to perform administrative tasks on your MinIO deployments.

```
service     restart and stop all MinIO servers
update      update all MinIO servers
info        display MinIO server information
user        manage users
group       manage groups
policy      manage policies defined in the MinIO server
config      manage MinIO server configuration
heal        heal disks, buckets and objects on MinIO server
profile     generate profile data for debugging purposes
top         provide top like statistics for MinIO
trace       show http trace for MinIO server
console     show console logs for MinIO server
prometheus  manages prometheus config
kms         perform KMS management operations
bucket      manage buckets defined in the MinIO server

```

## 1.  Download MinIO Client
### Docker Stable
```
docker pull minio/mc
docker run minio/mc admin info play
```

### Docker Edge
```
docker pull minio/mc:edge
docker run minio/mc:edge admin info server play
```

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

## 3. Add a MinIO Storage Service
MinIO server displays URL, access and secret keys.

#### Usage

```
mc alias set <ALIAS> <YOUR-MINIO-ENDPOINT> [YOUR-ACCESS-KEY] [YOUR-SECRET-KEY]
```

Keys must be supplied by argument or standard input.

<ALIAS> is simply a short name to your MinIO service. MinIO end-point, access and secret keys are supplied by your MinIO service. Admin API uses "S3v4" signature and cannot be changed.

### Examples

1. Keys by argument

   ```
   mc alias set minio http://192.168.1.51:9000 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
   ```

2. Keys by prompt

   ```
   mc alias set minio http://192.168.1.51:9000
   Enter Access Key: BKIKJAA5BMMU2RHO6IBB
   Enter Secret Key: V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
   ```

2. Keys by pipe

   ```
   echo -e "BKIKJAA5BMMU2RHO6IBB\nV7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12" | \
       mc alias set minio http://192.168.1.51:9000
   ```

## 4. Test Your Setup

*Example:*

Get MinIO server information for the configured alias `minio`

```
mc admin info minio
●  min.minio.io
   Uptime: 11 hours
   Version: 2020-01-17T22:08:02Z
   Network: 1/1 OK
   Drives: 4/4 OK

2.1 GiB Used, 158 Buckets, 12,092 Objects
4 drives online, 0 drives offline
```

## 5. Everyday Use
You may add shell aliases for info, healing.

```
alias minfo='mc admin info'
```

## 6. Global Options

### Option [--debug]
Debug option enables debug output to console.

*Example: Display verbose debug output for `info` command.*

```
mc: <DEBUG> GET /minio/admin/v2/info HTTP/1.1
Host: play.minio.io
User-Agent: MinIO (linux; amd64) madmin-go/0.0.1 mc/DEVELOPMENT.GOGET
Authorization: AWS4-HMAC-SHA256 Credential=**REDACTED**/20200120//s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=**REDACTED**
X-Amz-Content-Sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
X-Amz-Date: 20200120T185844Z
Accept-Encoding: gzip

mc: <DEBUG> HTTP/1.1 200 OK
Content-Length: 1105
Accept-Ranges: bytes
Connection: keep-alive
Content-Security-Policy: block-all-mixed-content
Content-Type: application/json
Date: Mon, 20 Jan 2020 18:58:44 GMT
Server: nginx/1.10.3 (Ubuntu)
Vary: Origin
X-Amz-Bucket-Region: us-east-1
X-Amz-Request-Id: 15EBAD6087210B2A
X-Xss-Protection: 1; mode=block

mc: <DEBUG> Response Time:  381.860854ms

●  play.minio.io
   Uptime: 11 hours
   Version: 2020-01-17T22:08:02Z
   Network: 1/1 OK
   Drives: 4/4 OK

2.1 GiB Used, 158 Buckets, 12,092 Objects
4 drives online, 0 drives offline
```

### Option [--json]
JSON option enables parseable output in [JSON lines](http://jsonlines.org/) format.

*Example: MinIO server information.*

```
mc admin --json info play
{
    "status": "success",
    "info": {
        "mode": "online",
        "region": "us-east-1",
        "deploymentID": "728e91fd-ed0c-4500-b13d-d143561518bf",
        "buckets": {
            "count": 158
        },
        "objects": {
            "count": 12092
        },
        "usage": {
            "size": 2249526349
        },
        "services": {
            "vault": {
                "status": "KMS configured using master key"
            },
            "ldap": {}
        },
        "backend": {
            "backendType": "Erasure",
            "onlineDisks": 4,
            "rrSCData": 2,
            "rrSCParity": 2,
            "standardSCData": 2,
            "standardSCParity": 2
        },
        "servers": [
            {
                "state": "ok",
                "endpoint": "play.minio.io",
                "uptime": 41216,
                "version": "2020-01-17T22:08:02Z",
                "commitID": "b0b25d558e25608e3a604888a0a43e58e8301dfb",
                "network": {
                    "play.minio.io": "online"
                },
                "disks": [
                    {
                        "path": "/home/play/data1",
                        "state": "ok",
                        "uuid": "c1f8dbf8-39c8-46cd-bab6-2c87d18db06a",
                        "totalspace": 8378122240,
                        "usedspace": 1410588672
                    },
                    {
                        "path": "/home/play/data2",
                        "state": "ok",
                        "uuid": "9616d28f-5f4d-47f4-9c6d-4deb0da07cad",
                        "totalspace": 8378122240,
                        "usedspace": 1410588672
                    },
                    {
                        "path": "/home/play/data3",
                        "state": "ok",
                        "uuid": "4c822d68-4d9a-4fa3-aabb-5bf5a58e5848",
                        "totalspace": 8378122240,
                        "usedspace": 1410588672
                    },
                    {
                        "path": "/home/play/data4",
                        "state": "ok",
                        "uuid": "95b5a33c-193b-4a11-b13a-a99bc1483182",
                        "totalspace": 8378122240,
                        "usedspace": 1410588672
                    }
                ]
            }
        ]
    }
}
```

### Option [--no-color]
This option disables the color theme. It is useful for dumb terminals.

### Option [--quiet]
Quiet option suppress chatty console output.

### Option [--config-dir]
Use this option to set a custom config path.

### Option [ --insecure]
Skip SSL certificate verification.

## 7. Commands

| Commands                                                               |
|:-----------------------------------------------------------------------|
| [**service** - restart and stop all MinIO servers](#service)           |
| [**update** - updates all MinIO servers](#update)                      |
| [**info** - display MinIO server information](#info)                   |
| [**user** - manage users](#user)                                       |
| [**group** - manage groups](#group)                                    |
| [**policy** - manage canned policies](#policy)                         |
| [**config** - manage server configuration file](#config)               |
| [**heal** - heal disks, buckets and objects on MinIO server](#heal)    |
| [**profile** - generate profile data for debugging purposes](#profile) |
| [**top** - provide top like statistics for MinIO](#top)                |
| [**trace** - show http trace for MinIO server](#trace)                 |
| [**console** - show console logs for MinIO server](#console)           |
| [**prometheus** - manages prometheus config settings](#prometheus)     |
| [**bucket** - manages buckets defined in the MinIO server](#bucket)     |

<a name="update"></a>
### Command `update` - updates all MinIO servers
`update` command provides a way to update all MinIO servers in a cluster. You can also use a private mirror server with `update` command to update your MinIO cluster. This is useful in cases where MinIO is running in an environment that doesn't have Internet access.

*Example: Update all MinIO servers.*
```
mc admin update play
Server `play` updated successfully from RELEASE.2019-08-14T20-49-49Z to RELEASE.2019-08-21T19-59-10Z
```

#### Steps to update MinIO using a private mirror
For using `update` command with private mirror server, you need to mirror the directory structure on `https://dl.minio.io/server/minio/release/linux-amd64/` on your private mirror server and then provide:

```
mc admin update myminio https://myfavorite-mirror.com/minio-server/linux-amd64/minio.sha256sum
Server `myminio` updated successfully from RELEASE.2019-08-14T20-49-49Z to RELEASE.2019-08-21T19-59-10Z
```

> NOTE:
> - An alias pointing to a distributed setup this command will automatically update all MinIO servers in the cluster.
> - `update` is a disruptive operation for your MinIO service, any on-going API operations will be forcibly canceled. So, it should be used only when you are planning MinIO upgrades for your deployment.
> - It is recommended to perform a restart after `update` successfully completes.

<a name="service"></a>
### Command `service` - restart and stop all MinIO servers
`service` command provides a way to restart and stop all MinIO servers.

> NOTE:
> - An alias pointing to a distributed setup this command will automatically execute the same actions across all servers.
> - `restart` and `stop` sub-commands are disruptive operations for your MinIO service, any on-going API operations will be forcibly canceled. So, it should be used only under administrative circumstances. Please use it with caution.

```
NAME:
  mc admin service - restart and stop all MinIO servers

FLAGS:
  --help, -h                       show help

COMMANDS:
  restart  restart all MinIO servers
  stop     stop all MinIO servers
```

*Example: Restart all MinIO servers.*
```
mc admin service restart play
Restarted `play` successfully.
```

<a name="info"></a>
### Command `info` - Display MinIO server information
`info` command displays server information of one or many MinIO servers (under distributed cluster)

```
NAME:
  mc admin info - get MinIO server information

FLAGS:
  --help, -h                       show help
```

*Example: Display MinIO server information.*

```
mc admin info play
●  play.minio.io
   Uptime: 11 hours
   Version: 2020-01-17T22:08:02Z
   Network: 1/1 OK
   Drives: 4/4 OK

2.1 GiB Used, 158 Buckets, 12,092 Objects
4 drives online, 0 drives offline
```

<a name="policy"></a>
### Command `policy` - Manage canned policies
`policy` command to add, remove, list policies, get info on a policy and to set a policy for a user on MinIO server.

```
NAME:
  mc admin policy - manage policies

FLAGS:
  --help, -h                       show help

COMMANDS:
  add      add new policy
  remove   remove policy
  list     list all policies
  info     show info on a policy
  set      set IAM policy on a user or group
```

*Example: List all canned policies on MinIO.*
```
mc admin policy list myminio/
diagnostics
readonly
readwrite
writeonly
```


*Example: Add a new policy 'listbucketsonly' on MinIO, with policy from /tmp/listbucketsonly.json.*
*When this policy is applied on a user, that user can only list the top layer buckets, but nothing else, no prefixes, no objects.*

*First create the json file, /tmp/listbucketsonly.json, with the following information.*
```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets"
      ],
      "Resource": [
        "arn:aws:s3:::*"
      ]
    }
  ]
}
```

*Add the policy as 'listbucketsonly' to the policy database*
```
mc admin policy add myminio/ listbucketsonly /tmp/listbucketsonly.json
Added policy `listbucketsonly` successfully.
```

*Example: Remove policy 'listbucketsonly' on MinIO.*

```
mc admin policy remove myminio/ listbucketsonly
Removed policy `listbucketsonly` successfully.
```

*Example: Show info on a canned policy, 'writeonly'*

```
mc admin policy info myminio/ writeonly
{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:PutObject"],"Resource":["arn:aws:s3:::*"]}]}
```

*Example: Set the canned policy.'writeonly' on a user or group*

```
mc admin policy set myminio/ writeonly user=someuser
Policy writeonly is set on user `someuser`

mc admin policy set myminio/ writeonly group=somegroup
Policy writeonly is set on group `somegroup`
```

<a name="user"></a>
### Command `user` - Manage users
`user` command to add, remove, enable, disable, list users on MinIO server.

```
NAME:
  mc admin user - manage users

FLAGS:
  --help, -h                       show help

COMMANDS:
  add      add new user
  disable  disable user
  enable   enable user
  remove   remove user
  list     list all users
  info     display info of a user
```

*Example: Add a new user 'newuser' on MinIO.*

```
mc admin user add myminio/ newuser newuser123
```

*Example: Add a new user 'newuser' on MinIO, using standard input.*

```
mc admin user add myminio/
Enter Access Key: newuser
Enter Secret Key: newuser123
```

*Example: Disable a user 'newuser' on MinIO.*

```
mc admin user disable myminio/ newuser
```

*Example: Enable a user 'newuser' on MinIO.*

```
mc admin user enable myminio/ newuser
```

*Example: Remove user 'newuser' on MinIO.*

```
mc admin user remove myminio/ newuser
```

*Example: List all users on MinIO.*

```
mc admin user list --json myminio/
{"status":"success","accessKey":"newuser","userStatus":"enabled"}
```

*Example: Display info of a user*

```
mc admin user info myminio someuser
```

<a name="group"></a>
### Command `group` - Manage groups
`group` command to add, remove, info, list, enable, disable groups on MinIO server.

```
NAME:
  mc admin group - manage groups

USAGE:
  mc admin group COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  add      add users to a new or existing group
  remove   remove group or members from a group
  info     display group info
  list     display list of groups
  enable   Enable a group
  disable  Disable a group
```

*Example: Add a pair of users to a group 'somegroup' on MinIO.*

Group is created if it does not exist.

```
mc admin group add myminio somegroup someuser1 someuser2
```

*Example: Remove a pair of users from a group 'somegroup' on MinIO.*

```
mc admin group remove myminio somegroup someuser1 someuser2
```

*Example: Remove a group 'somegroup' on MinIO.*

Only works if the given group is empty.

```
mc admin group remove myminio somegroup
```

*Example: Get info on a group 'somegroup' on MinIO.*

```
mc admin group info myminio somegroup
```

*Example: List all groups on MinIO.*

```
mc admin group list myminio
```

*Example: Enable a group 'somegroup' on MinIO.*

```
mc admin group enable myminio somegroup
```

*Example: Disable a group 'somegroup' on MinIO.*

```
mc admin group disable myminio somegroup
```

<a name="config"></a>
### Command `config` - Manage server configuration
`config` command to manage MinIO server configuration.

```
NAME:
  mc admin config - manage configuration file

USAGE:
  mc admin config COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  get     get config of a MinIO server/cluster.
  set     set new config file to a MinIO server/cluster.

FLAGS:
  --help, -h                       Show help.
```

*Example: Get server configuration of a MinIO server/cluster.*

```
mc admin config get myminio > /tmp/my-serverconfig
```

*Example: Set server configuration of a MinIO server/cluster.*

```
mc admin config set myminio < /tmp/my-serverconfig
```

<a name="heal"></a>
### Command `heal` - Heal disks, buckets and objects on MinIO server
Healing is automatic on server side which runs on a continuous basis on a low priority thread, `mc admin heal` is deprecated and will be removed in future.

<a name="profile"></a>
### Command `profile` - generate profile data for debugging purposes

```
NAME:
  mc admin profile - generate profile data for debugging purposes

COMMANDS:
  start  start recording profile data
  stop   stop and download profile data
```

Start CPU profiling
```
mc admin profile start --type cpu myminio/
```

<a name="top"></a>
### Command `top` - provide top like statistics for MinIO
NOTE: This command is only applicable for a distributed MinIO setup. It is not supported on single node and gateway deployments.

```
NAME:
  mc admin top - provide top like statistics for MinIO

COMMANDS:
  locks  Get a list of the 10 oldest locks on a MinIO cluster.
```

*Example: Get a list of the 10 oldest locks on a distributed MinIO cluster, where 'myminio' is the MinIO cluster alias.*

```
mc admin top locks myminio
```

<a name="trace"></a>
### Command `trace` - Show http trace for MinIO server
`trace` command displays server http trace of one or all MinIO servers (under distributed cluster)

```sh
NAME:
  mc admin trace - show http trace for MinIO server

FLAGS:
  --verbose, -v                 print verbose trace
  --all, -a                     trace all traffic (including internode traffic between MinIO servers)
  --errors, -e                  trace failed requests only
  --help, -h                    show help
```

*Example: Display MinIO server http trace.*

```sh
mc admin trace myminio
172.16.238.1 [REQUEST (objectAPIHandlers).ListBucketsHandler-fm] [154828542.525557] [2019-01-23 23:17:05 +0000]
172.16.238.1 GET /
172.16.238.1 Host: 172.16.238.3:9000
172.16.238.1 X-Amz-Date: 20190123T231705Z
172.16.238.1 Authorization: AWS4-HMAC-SHA256 Credential=minio/20190123/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=8385097f264efaf1b71a9b56514b8166bb0a03af8552f83e2658f877776c46b3
172.16.238.1 User-Agent: MinIO (linux; amd64) minio-go/v7.0.8 mc/2019-01-23T23:15:38Z
172.16.238.1 X-Amz-Content-Sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
172.16.238.1
172.16.238.1 <BODY>
172.16.238.1 [RESPONSE] [154828542.525557] [2019-01-23 23:17:05 +0000]
172.16.238.1 200 OK
172.16.238.1 X-Amz-Request-Id: 157C9D641F42E547
172.16.238.1 X-Minio-Deployment-Id: 5f20fd91-6880-455f-a26d-07804b6821ca
172.16.238.1 X-Xss-Protection: 1; mode=block
172.16.238.1 Accept-Ranges: bytes
172.16.238.1 Content-Security-Policy: block-all-mixed-content
172.16.238.1 Content-Type: application/xml
172.16.238.1 Server: MinIO/RELEASE.2019-09-05T23-24-38Z
172.16.238.1 Vary: Origin
...
```

<a name="console"></a>
### Command `console` - show console logs for MinIO server
`console` command displays server logs of one or all MinIO servers (under distributed cluster)

```sh
NAME:
  mc admin console - show console logs for MinIO server

FLAGS:
  --limit value, -l value       show last n log entries (default: 10)
  --help, -h                    show help
```

*Example: Display MinIO server http trace.*

```sh
mc admin console myminio

 API: SYSTEM(bucket=images)
 Time: 22:48:06 PDT 09/05/2019
 DeploymentID: 6faeded5-5cf3-4133-8a37-07c5d500207c
 RequestID: <none>
 RemoteHost: <none>
 UserAgent: <none>
 Error: ARN 'arn:minio:sqs:us-east-1:1:webhook' not found
        4: cmd/notification.go:1189:cmd.readNotificationConfig()
        3: cmd/notification.go:780:cmd.(*NotificationSys).refresh()
        2: cmd/notification.go:815:cmd.(*NotificationSys).Init()
        1: cmd/server-main.go:375:cmd.serverMain()
```

<a name="prometheus"></a>

### Command `prometheus` - Manages prometheus config settings

`generate` command generates the prometheus config (To be pasted in `prometheus.yml`)

```sh
NAME:
  mc admin prometheus - manages prometheus config

USAGE:
  mc admin prometheus COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  generate  generates prometheus config

```

_Example: Generates prometheus config for an <alias>._

```sh
mc admin prometheus generate <alias>
- job_name: minio-job
  bearer_token: <token>
  metrics_path: /minio/prometheus/metrics
  scheme: http
  static_configs:
  - targets: ['localhost:9000']
```

<a name="kms"></a>

### Command `kms` - perform KMS management operations

The `kms` command can be used to perform KMS management operations.

```sh
NAME:
  mc admin kms - perform KMS management operations

USAGE:
  mc admin kms COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]
```

The `key` sub-command can be used to perform master key management operations.

```sh
NAME:
  mc admin kms key - manage KMS keys

USAGE:
  mc admin kms key COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]
```

*Example: Display status information for the default master key*

```sh
mc admin kms key status play
Key: my-minio-key
 	 • Encryption ✔
 	 • Decryption ✔
```

*Example: Create a new master key at the KMS*

```sh
mc admin kms key create play my-key

Created master key `my-key` successfully
```

*Example: Display status information for one particular master key*

```sh
mc admin kms key status play my-key
Key: my-key
 	 • Encryption ✔
 	 • Decryption ✔
```
<a name = "bucket"></a>
<a name="quota"></a>
### Command `quota` - Set/Get bucket quota
`quota` command to set or get bucket quota on MinIO server.

```
NAME:
  mc admin bucket quota - manage bucket quota

USAGE:
  mc admin bucket quota TARGET [--fifo QUOTA | --hard QUOTA | --clear]

QUOTA
  quota accepts human-readable case-insensitive number
  suffixes such as "k", "m", "g" and "t" referring to the metric units KB,
  MB, GB and TB respectively. Adding an "i" to these prefixes, uses the IEC
  units, so that "gi" refers to "gibibyte" or "GiB". A "b" at the end is
  also accepted. Without suffixes the unit is bytes.

```
*Example: List bucket quota on bucket 'mybucket' on MinIO.*

```
mc admin bucket quota myminio/mybucket
```

*Example: Set a hard bucket quota of 64Mb for bucket 'mybucket' on MinIO.*

```
mc admin bucket quota myminio/mybucket --hard 64MB
```

*Example: Reset bucket quota configured for bucket 'mybucket' on MinIO.*

```
mc admin bucket quota myminio/mybucket --clear
```

<a name="remote"></a>
### Command `remote` - configure remote target buckets
`remote` command manages remote bucket targets on MinIO server.

```
NAME:
  mc admin bucket remote - configure remote bucket targets

USAGE:
  mc admin bucket remote COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  add  add a new remote target
  ls   list remote target ARN(s)
  rm   remove configured remote target

```
*Example: Add a new replication target `targetbucket` in region `us-west-1` on `https://minio2:9000` for bucket `srcbucket` on MinIO server. `foobar` and `foo12345` are credentials to target endpoint.
*

```
mc admin bucket remote add myminio/srcbucket https://foobar:foobar12345@minio2:9000/targetbucket --service "replication" --region "us-west-1"
ARN = `arn:minio:replication:us-west-1:1f8712ba-e38f-4429-bcb1-a7bb5aa97447:targetbucket`
```

*Example: Get remote target for replication on bucket 'srcbucket' in MinIO.*

```
mc admin bucket remote ls myminio/srcbucket --service "replication"
```

*Example: List remote target(s) on bucket 'srcbucket' in MinIO.*

```
mc admin bucket remote ls myminio/srcbucket
```

*Example: List remote target(s) on MinIO.*

```
mc admin bucket remote ls myminio
```

*Example: Remove bucket target configured for bucket 'srcbucket' on MinIO with arn `arn:minio:replication:us-west-1:1f8712ba-e38f-4429-bcb1-a7bb5aa97447:targetbucket`.*

```
mc admin bucket remote rm myminio/srcbucket --arn "arn:minio:replication:us-west-1:1f8712ba-e38f-4429-bcb1-a7bb5aa97447:targetbucket"
```
