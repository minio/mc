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

### Option [--version]
Display the current version of `mc` installed

*Example: Print version of mc.*

```
mc --version
mc version RELEASE.2020-04-25T00-43-23Z
```

## 7. Commands

| Commands                                                                           |
|:-----------------------------------------------------------------------------------|
| [**service** - restart and stop all MinIO servers](#service)                       |
| [**update** - updates all MinIO servers](#update)                                  |
| [**info** - display MinIO server information](#info)                               |
| [**user** - manage users](#user)                                                   |
| [**group** - manage groups](#group)                                                |
| [**policy** - manage canned policies](#policy)                                     |
| [**replicate** - manage MinIO site replication](#replicate)                        |
| [**idp** - manage MinIO IDentity Provider server configuration](#idp)              |
| [**config** - manage server configuration file](#config)                           |
| [**decommission, decom** - manage MinIO server pool decommissioning](#config)      |
| [**heal** - heal bucket(s) and object(s) on MinIO server](#heal)                   |
| [**prometheus** - manages prometheus config settings](#prometheus)                 |
| [**kms** - perform KMS management operations](#kms)                                |
| [**bucket** - manages buckets defined in the MinIO server](#bucket)                |
| [**scanner** - provide MinIO scanner info](#scanner)                               |
| [**top** - provide top like statistics for MinIO](#top)                            |
| [**trace** - show http trace for MinIO server](#trace)                             |
| [**logs** - show MinIO logs](#logs)                                                |
| [**cluster** - manage MinIO cluster metadata](#cluster)                            |
| [**rebalance** - Manage MinIO rebalance](#rebalance)                               |

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
  --help, -h                    show help
  
COMMANDS:
  restart  restart a MinIO cluster
  stop     stop a MinIO cluster
  unfreeze unfreeze S3 API calls on MinIO cluster
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
  mc admin info - display MinIO server information

FLAGS:
  --help, -h                    show help
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
  mc admin policy - manage policies defined in the MinIO server

FLAGS:
  --help, -h                    show help

COMMANDS:
  create    create a new IAM policy
  remove    remove an IAM policy
  list      list all IAM policies
  info      show info on an IAM policy
  attach    attach an IAM policy to a user or group
  detach    detach an IAM policy from a user or group
  entities  list policy association entities
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
mc admin policy create myminio/ listbucketsonly /tmp/listbucketsonly.json
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

*Example: Attach the canned policy.'writeonly' on a user or group*

```
mc admin policy attach myminio/ writeonly user=someuser
Policy `writeonly` successfully attached to user `someuser`
```

*Example: Detach the canned policy.'writeonly' on a user or group*

```
mc admin policy detach myminio/ writeonly group=somegroup
Policy `writeonly` successfully detached from group `somegroup`
```

<a name="user"></a>
### Command `user` - Manage users
`user` command to add, remove, enable, disable, list users on MinIO server.

```
NAME:
  mc admin user - manage users

FLAGS:
  --help, -h                    show help

COMMANDS:
  add      add a new user
  disable  disable user
  enable   enable user
  remove   remove user
  list     list all users
  info     display info of a user
  policy   export user policies in JSON format
  svcacct  manage service accounts
  sts      manage STS accounts
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

<a name="replicate"></a>
### Command `replicate` - manage MinIO site replication
`replicate` command to add, update, rm sites for replication.

```
NAME:
  mc admin replicate - manage MinIO site replication

FLAGS:
  --help, -h                    show help

COMMANDS:
  add     add one or more sites for replication
  update  modify endpoint of site participating in site replication
  rm      remove one or more sites from site replication
  info    get site replication information
  status  display site replication status
  resync  resync content to site
```

*Example: Add a site for cluster-level replication.*

```
mc admin replicate add minio1 minio2
```

*Example: Edit a site endpoint participating in cluster-level replication.*

```
mc admin replicate update myminio --deployment-id c1758167-4426-454f-9aae-5c3dfdf6df64 --endpoint https://minio2:9000
```

*Example: Remove site replication for all sites.*

```
mc admin replicate rm minio2 --all --force
```

*Example: Remove site replication for site with site names alpha, baker from active cluster minio2.*

```
mc admin replicate rm minio2 alpha baker --force
```

*Example: Get Site Replication information.*

```
mc admin replicate info minio1
```

*Example: Display overall site replication status.*

```
mc admin replicate status minio1
```

*Example: Resync bucket data from minio1 to minio2.*

```
mc admin replicate resync start minio1 minio2
```

*Example: Display status of resync from minio1 to minio2.*

```
mc admin replicate resync status minio1 minio2
```

*Example: Cancel ongoing resync of bucket data from minio1 to minio2.*

```
mc admin replicate resync cancel minio1 minio2
```


<a name="idp"></a>
### Command `idp` - manage MinIO IDentity Provider server configuration
`idp` command to add, update, remove, list, enable, disable OpenID or Ldap IDP server configuration.

```
NAME:
  mc admin idp - manage MinIO IDentity Provider server configuration

FLAGS:
  --help, -h                    show help

COMMANDS:
  openid  manage OpenID IDP server configuration
  ldap    manage Ldap IDP server configuration
```

*Example: Create OpenID IDP configuration named "dex_test".*

```
mc admin idp openid add play/ dex_test \
client_id=minio-client-app \
client_secret=minio-client-app-secret \
config_url="http://localhost:5556/dex/.well-known/openid-configuration" \
scopes="openid,groups" \
redirect_uri="http://127.0.0.1:10000/oauth_callback" \
role_policy="consoleAdmin"
```

*Example: Update configuration for OpenID IDP configuration named "dex_test".*

```
mc admin idp openid update play/ dex_test \
scopes="openid,groups" \
role_policy="consoleAdmin"
```

*Example: Remove OpenID IDP configuration named "dex_test".*

```
mc admin idp openid remove play/ dex_test
```

*Example:  List configurations for OpenID IDP.*

```
mc admin idp openid list play/
```

*Example: Get configuration info on OpenID IDP configuration named "dex_test".*

```
mc admin idp openid info play/ dex_test
```

*Example: Enable OpenID IDP configuration named "dex_test".*

```
mc admin idp openid enable play/ dex_test
```

*Example: Disable OpenID IDP configuration named "dex_test".*

```
mc admin idp openid disable play/ dex_test
```

*Example: Create LDAP IDentity Provider configuration.*

```
mc admin idp ldap add myminio/ \
server_addr=myldapserver:636 \
lookup_bind_dn=cn=admin,dc=min,dc=io \
lookup_bind_password=somesecret \
user_dn_search_base_dn=dc=min,dc=io \
user_dn_search_filter="(uid=%s)" \
group_search_base_dn=ou=swengg,dc=min,dc=io \
group_search_filter="(&(objectclass=groupofnames)(member=%d))"
```

*Example: Remove the default LDAP IDP configuration.*

```
mc admin idp ldap remove play/
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
  get      interactively retrieve a config key parameters
  set      interactively set a config key parameters
  reset    interactively reset a config key parameters
  history  show all historic configuration changes
  restore  rollback back changes to a specific config history
  export   export all config keys to STDOUT
  import   import multiple config keys from STDIN

FLAGS:
  --help, -h                    show help
```

*Example: Get 'etcd' sub-system configuration.*

```
mc admin config get myminio etcd
etcd endpoints= path_prefix= coredns_path=/skydns client_cert= client_cert_key=
```

*Example: Set specific settings on 'etcd' sub-system.*
```
mc admin config set myminio etcd endpoints=http://etcd.svc.cluster.local:2379
```

*Example: Get entire server configuration of a MinIO server/cluster.*

```
mc admin config export myminio > /tmp/my-serverconfig
```

*Example: Set entire server configuration of a MinIO server/cluster.*

```
mc admin config import myminio < /tmp/my-serverconfig
```

<a name="decommission"></a>
### Command `decommission` - Manage MinIO server pool decommissioning
`decommission` manage MinIO server pool decommissioning.

```
NAME:
  mc admin decommission - manage MinIO server pool decommissioning

USAGE:
  mc admin decommission COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  start   start decommissioning a pool
  status  show current decommissioning status
  cancel  cancel an ongoing decommissioning of a pool

FLAGS:
  --help, -h                    show help
```

*Example: Start decommissioning a pool for removal.*

```
mc admin decommission start myminio/ http://server{5...8}/disk{1...4}
```

*Example: Show current decommissioning status.*
```
mc admin decommission status myminio/ http://server{5...8}/disk{1...4}
```

*Example: List all current decommissioning status of all pools.*

```
mc admin decommission status myminio/
```

*Example: Cancel an ongoing decommissioning of a pool.*

```
mc admin decommission cancel myminio/ http://server{5...8}/disk{1...4}
```

*Example: Cancel all decommissioning of a pool.*

```
mc admin decommission cancel myminio/
```

<a name="heal"></a>
### Command `heal` - heal bucket(s) and object(s) on MinIO server
Healing is automatic on server side which runs on a continuous basis on a low priority thread.

<a name="heal"></a>
### Command `heal` - heal bucket(s) and object(s) on MinIO server
Healing is automatic on server side which runs on a continuous basis on a low priority thread.

```
NAME:
  mc admin heal - heal bucket(s) and object(s) on MinIO server

USAGE:
  mc admin heal [FLAGS] TARGET
  
FLAGS:
  --help, -h                    show help
```

*Example: Monitor healing status on a running server at alias 'myminio'.*

```
 mc admin heal myminio/
```

<a name="trace"></a>
### Command `trace` - Show http trace for MinIO server
`trace` command displays server http trace of one or all MinIO servers (under distributed cluster)

```sh
NAME:
  mc admin trace - show http trace for MinIO server

FLAGS:
  --verbose, -v                 print verbose trace
  --all, -a                     trace all call types
  --call value                  trace only matching call types. See CALL TYPES below for list. (default: s3)
  --status-code value           trace only matching status code
  --method value                trace only matching HTTP method
  --funcname value              trace only matching func name
  --path value                  trace only matching path
  --node value                  trace only matching servers
  --request-header value        trace only matching request headers
  --errors, -e                  trace only failed requests
  --filter-request              trace calls only with request bytes greater than this threshold, use with filter-size
  --filter-response             trace calls only with response bytes greater than this threshold, use with filter-size
  --response-duration 5ms       trace calls only with response duration greater than this threshold (e.g. 5ms) (default: 0s)
  --filter-size value           filter size, use with filter (see UNITS)
  --help, -h                    show help
  
CALL TYPES:
  batch-replication:   Trace Batch Replication (alias: brep)
  bootstrap:           Trace Bootstrap operations
  decommission:        Trace Decommission operations (alias: decom)
  healing:             Trace Healing operations (alias: heal)
  internal:            Trace Internal RPC calls
  os:                  Trace Operating System calls
  rebalance:           Trace Server Pool Rebalancing operations
  replication-resync:  Trace Replication Resync operations (alias: resync)
  s3:                  Trace S3 API calls
  scanner:             Trace Scanner calls
  storage:             Trace Storage calls

UNITS
  --filter-size flags use with --filter-response or --filter-request accept human-readable case-insensitive number
  suffixes such as "k", "m", "g" and "t" referring to the metric units KB,
  MB, GB and TB respectively. Adding an "i" to these prefixes, uses the IEC
  units, so that "gi" refers to "gibibyte" or "GiB". A "b" at the end is
  also accepted. Without suffixes the unit is bytes.

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

*Example: Show verbose console trace for MinIO server.*

```
 mc admin trace -v -a myminio
```

*Example: Show trace only for failed requests for MinIO server.*

```
 mc admin trace -v -e myminio
```

*Example: Show verbose console trace for requests with '503' status code.*

```
 mc admin trace -v --status-code 503 myminio
```

*Example: Show console trace for a specific path.*

```
 mc admin trace --path my-bucket/my-prefix/* myminio
```

*Example: Show console trace for requests with '404' and '503' status code.*

```
 mc admin trace --status-code 404 --status-code 503 myminio
```

*Example: Show trace only for requests bytes greater than 1MB.*

```
 mc admin trace --filter-request --filter-size 1MB myminio
```

*Example: Show trace only for response bytes greater than 1MB.*

```
 mc admin trace --filter-response --filter-size 1MB myminio
```

*Example: Show trace only for requests operations duration greater than 5ms.*

```
 mc admin trace --response-duration 5ms myminio
```

<a name="scanner"></a>
### Command `scanner` - Provide MinIO scanner info
`scanner` provide MinIO scanner info.

```sh
NAME:
  mc admin scanner - provide MinIO scanner info

FLAGS:
  --help, -h                    show help
```

*Example: Show scanner trace for MinIO server.*

```
 mc admin scanner trace myminio
```

*Example: Display current in-progress all scanner operations.*

```
 mc admin scanner status myminio/
```

<a name="console"></a>
### Command `console` - show console logs for MinIO server
This command is deprecated and will be removed in a future release. Use 'mc support logs show' instead.

<a name="logs"></a>
### Command `logs` - Show MinIO logs
`logs` show console logs for MinIO server.

```
NAME:
  mc admin logs - show MinIO logs
USAGE:
  mc admin logs [FLAGS] TARGET [NODENAME]

FLAGS:
  --last value, -l value        show last n log entries (default: 10)
  --type value, -t value        list error logs by type. Valid options are '[minio, application, all]' (default: "all")
  --help, -h                    show help
```

*Example: Show logs for a MinIO server with alias 'myminio'.*

```
 mc admin logs myminio
```

*Example: Show last 5 log entries for node 'node1' for a MinIO server with alias 'myminio'.*

```
 mc admin logs --last 5 myminio node1
```

*Example: Show application errors in logs for a MinIO server with alias 'myminio'.*

```
 mc admin logs --type application myminio
```

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

<a name="rebalance"></a>
### Command `rebalance` - Manage MinIO rebalance
`rebalance` manage MinIO rebalance.

```
NAME:
  mc admin rebalance - Manage MinIO rebalance

USAGE:
  mc admin rebalance COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

FLAGS:
  --help, -h                    show help
```

*Example: Start rebalance on a MinIO deployment with alias myminio.*

```
 mc admin rebalance start myminio
```

*Example: Stop an ongoing rebalance on a MinIO deployment with alias myminio.*

```
 mc admin rebalance stop myminio
```

*Example: Summarize ongoing rebalance on a MinIO deployment with alias myminio.*

```
 mc admin rebalance status myminio
```

<a name="prometheus"></a>

### Command `prometheus` - Manages prometheus config settings

`generate` command generates the prometheus config (To be pasted in `prometheus.yml`)
`metrics` command print cluster wide prometheus metrics

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
  metrics_path: /minio/v2/metrics/cluster
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
This command is deprecated and will be removed in a future release. Use 'mc quota set|info|clear' instead.
<a name="remote"></a>
### Command `remote` - configure remote target buckets
`remote` command manages remote bucket targets on MinIO server.

```
NAME:
  mc admin bucket remote - configure remote bucket targets 

This command is deprecated and will be removed in a future release. Use 'mc replicate add|update|rm` commands to manage remote targets
