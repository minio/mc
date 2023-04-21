# MinIO Admin Complete Guide [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io)

MinIO Client (mc) provides `admin` sub-command to perform administrative tasks on your MinIO deployments.

```
service              restart and stop all MinIO servers
update               update all MinIO servers
info                 display MinIO server information
user                 manage users
group                manage groups
policy               manage policies defined in the MinIO server
replicate            manage MinIO site replication
idp                  manage MinIO IDentity Provider server configuration
config               manage MinIO server configuration
decommission, decom  manage MinIO server pool decommissioning
heal                 heal bucket(s) and object(s) on MinIO server
prometheus           manages prometheus config
kms                  perform KMS management operations
bucket               manage buckets defined in the MinIO server
tier                 manage remote tier targets for ILM transition
scanner              provide MinIO scanner info
top                  provide top like statistics for MinIO
trace                show http trace for MinIO server
cluster              manage MinIO cluster metadata
rebalance            Manage MinIO rebalance
logs                 show MinIO logs
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
            "rrSCParity": 2,
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

| Commands                                                                                                                                                       |
|:---------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [**service** - restart and stop all MinIO servers](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-service.html)                             |
| [**update** - updates all MinIO servers](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-update.html)                                        |
| [**info** - display MinIO server information](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-info.html)                                     |
| [**user** - manage users](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-user.html)                                                         |
| [**group** - manage groups](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-group.html)                                                      |
| [**policy** - manage canned policies](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-policy.html)                                           |
| [**replicate** - manage MinIO site replication](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-replicate.html)                              |
| [**idp** - manage MinIO IDentity Provider server configuration](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-idp-ldap.html)               |
| [**config** - manage server configuration file](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-config.html)                                 |
| [**decommission, decom** - manage MinIO server pool decommissioning](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-decommission.html)      |
| [**heal** - heal bucket(s) and object(s) on MinIO server](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-heal.html)                         |
| [**prometheus** - manages prometheus config settings](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-prometheus.html)                       |
| [**kms** - perform KMS management operations](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-kms-key.html)                                  |
| [**bucket** - manages buckets defined in the MinIO server](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-bucket-remote.html)               |
| [**scanner** - provide MinIO scanner info](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-trace.html)                                       |
| [**top** - provide top like statistics for MinIO](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-top.html)                                  |
| [**trace** - show http trace for MinIO server](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-trace.html)                                   |
| [**logs** - show MinIO logs](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-logs.html)                                                      |
| [**cluster** - manage MinIO cluster metadata](#cluster)                                                                                                        |
| [**rebalance** - Manage MinIO rebalance](https://min.io/docs/minio/linux/reference/minio-mc-admin/mc-admin-rebalance.html)                                     |

<a name="cluster"></a>
### Command `cluster` - Manage MinIO cluster metadata
`cluster` manage MinIO cluster metadata.

```
NAME:
  mc admin cluster - manage MinIO cluster metadata

USAGE:
  mc admin cluster COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

FLAGS:
  --help, -h                    show help
```

*Example: Recover bucket metadata for all buckets from previously saved bucket metadata backup.*

```
 mc admin cluster bucket import myminio /backups/cluster-metadata.zip
```

*Example: Save metadata of all buckets to a zip file.*

```
 mc admin cluster bucket export myminio
```

*Example: Set IAM info from previously exported metadata zip file.*

```
 mc admin cluster iam import myminio /tmp/myminio-iam-info.zip
```

*Example: Download all IAM metadata for cluster into zip file.*

```
 mc admin cluster iam export myminio
```