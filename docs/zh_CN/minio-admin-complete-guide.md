# MinIO 管理完全指南 [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io)

MinIO Client (mc) 提供了`admin`子命令以在MinIO部署上执行管理任务.

```sh
service      停止，重启或得到MinIO服务器的状态
info         显示MinIO服务器信息
user         管理用户
policy       管理封装策略
config       管理配置文件
heal         恢复MinIO服务器上的磁盘，存储桶和对象
top          为MinIO提供最类似的统计数据
```

## 1.  下载MinIO客户端
### Docker 稳定版
```
docker pull minio/mc
docker run minio/mc admin info play
```

### Docker 尝鲜版
```
docker pull minio/mc:edge
docker run minio/mc:edge admin info play
```

### Homebrew (macOS)
使用[Homebrew](http://brew.sh/)安装mc包

```sh
brew install minio/stable/mc
mc --help
```

### 下载二进制文件 (GNU/Linux)
| 平台 | CPU架构 | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.minio.io/client/mc/release/linux-amd64/mc |
||64-bit PPC|https://dl.minio.io/client/mc/release/linux-ppc64le/mc |

```sh
chmod +x mc
./mc --help
```

### 下载二进制文件 (Microsoft Windows)
| Platform | Architecture | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit Intel|https://dl.minio.io/client/mc/release/windows-amd64/mc.exe |

```sh
mc.exe --help
```

### 源码安装
通过源码安装仅适用于开发人员和高级用户。`mc update`命令不支持基于源码安装的更新通知。请从[https://min.io/download/#minio-client](https://min.io/download/#minio-client)下载官方版本。

如果您没有正在工作的Golang环境，请参照[如何安装Golang](https://docs.min.io/docs/how-to-install-golang)。

```sh
go get -d github.com/minio/mc
cd ${GOPATH}/src/github.com/minio/mc
make
```

## 2. 运行 MinIO 客户端

### GNU/Linux

```sh
chmod +x mc
./mc --help
```

### macOS

```sh
chmod 755 mc
./mc --help
```

### Microsoft Windows

```sh
mc.exe --help
```

## 3. 添加MinIO存储服务
MinIO服务显示URL，访问（access）和密钥（secret key）。

#### 用法

```sh
mc config host add <ALIAS> <YOUR-MINIO-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY>
```

别名是MinIO服务的简称。MinIO 端点，访问和密钥是由你的MinIO服务提供的。Admin API使用"S3V4"签名而且不能被更改。

```sh
mc config host add minio http://192.168.1.51:9000 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

## 4. 设置测试

*示例:*

获取配置别名为`minio`的MinIO服务器信息

```sh
mc admin info minio

●  192.168.1.51:9000
   Uptime : online since 1 day ago
  Version : 2018-05-28T04:31:38Z
   Region :
 SQS ARNs : <none>
    Stats : Incoming 82GiB, Outgoing 28GiB
  Storage : Used 7.4GiB
```

## 5. 日常使用
你可以为info，healing添加shell别名。

```sh
alias minfo='mc admin info'
alias mheal='mc admin heal'
```

## 6. 全局选项

### 选项 [--debug]
debug选项启用debug输出到控制台。

*示例: 显示`info`命令的详细调试输出.*

```sh
mc admin --debug info minio
mc: <DEBUG> GET /minio/admin/v1/info HTTP/1.1
Host: 192.168.1.51:9000
User-Agent: MinIO (linux; amd64) madmin-go/0.0.1 mc/2018-05-23T23:43:34Z
Authorization: AWS4-HMAC-SHA256 Credential=**REDACTED**/20180530/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=**REDACTED**
X-Amz-Content-Sha256: UNSIGNED-PAYLOAD
X-Amz-Date: 20180530T001808Z
Accept-Encoding: gzip

mc: <DEBUG> HTTP/1.1 200 OK
Transfer-Encoding: chunked
Accept-Ranges: bytes
Content-Security-Policy: block-all-mixed-content
Content-Type: application/json
Date: Wed, 30 May 2018 00:18:08 GMT
Server: MinIO/DEVELOPMENT.2018-05-28T04-31-38Z (linux; amd64)
Vary: Origin
X-Amz-Request-Id: 1533440573A63034
X-Xss-Protection: "1; mode=block"

mc: <DEBUG> Response Time:  140.70112ms

●  192.168.1.51:9000
   Uptime : online since 1 day ago
  Version : 2018-05-28T04:31:38Z
   Region :
 SQS ARNs : <none>
    Stats : Incoming 82GiB, Outgoing 28GiB
  Storage : Used 7.4GiB
```

### 选项 [--json]
JSON选项启用JSON形式的可解析输出。

*示例：MinIO服务信息。*

```sh
mc admin --json info minio
{
  "status": "success",
  "service": "on",
  "address": "192.168.1.51:9000",
  "error": "",
  "storage": {
    "used": 7979370172,
    "backend": {
      "backendType": "FS"
    }
  },
  "network": {
    "transferred": 90473434722,
    "received": 30662519192
  },
  "server": {
    "uptime": 157467244813288,
    "version": "2018-05-28T04:31:38Z",
    "commitID": "7d8c5ffb13334f4aec20a35bd2575bd7c740fb7a",
    "region": "",
    "sqsARN": []
  }
}
```

### 选项 [--no-color]
这个选项禁用颜色主题，对于简易终端非常有用。

### 选项 [--quiet]
Quiet选项可以抑制控制台大量信息的输出。

### 选项 [--config-dir]
使用这个选项去设置一个个性化的配置路径。

### 选项 [ --insecure]
跳过SSL证书验证。

## 7. 命令

|   |
|:---|
|[**service** - start, stop or get the status of MinIO server](#service) |
|[**info** - display MinIO server information](#info) |
|[**user** - manage users](#user) |
|[**policy** - manage canned policies](#policy) |
|[**config** - manage server configuration file](#config)|
|[**heal** - heal disks, buckets and objects on MinIO server](#heal) |
|[**top** - provide top like statistics for MinIO](#top) |

<a name="service"></a>

### 命令`service` - 停止，重启或得到minio服务器的状态

`service` 命令提供一个重启，停止或得到MinIO服务(分布式集群)状态的方式。

```sh
NAME:
  mc admin service - stop, restart or get status of minio server

FLAGS:
  --help, -h                       show help

COMMANDS:
  status   get the status of minio server
  restart  restart minio server
  stop     stop minio server
```

*示例: 显示MinIO服务器服务的运行时间*

```sh
mc admin service status play
Uptime: 1 days 19 hours 57 minutes 39 seconds.
```

*示例： 重启远程minio服务。*
注意： `restart`和`stop`子命令对于你的MinIO服务来说是极具破环性的操作，任何正在进行的API操作都将被强制取消。 因此，它应该只在某些特定情况下被使用。请一定要谨慎使用。

```sh
mc admin service restart play
Restarted `play` successfully.
```

<a name="info"></a>
### `info` 命令 - 显示MinIO服务器信息
`info` 命令显示一个或多个MinIO服务器(在分布式集群下)的服务器信息。 

```sh
NAME:
  mc admin info - get minio server information

FLAGS:
  --help, -h                       show help
```

*示例： 显示MinIO服务器信息*

```sh
mc admin info play
●  play.minio.io:9000
   Uptime : online since 1 day ago
  Version : 2018-05-28T04:31:38Z
   Region :
 SQS ARNs : <none>
    Stats : Incoming 82GiB, Outgoing 28GiB
  Storage : Used 8.2GiB
```

<a name="policy"></a>

### `policy`命令 - 管理封装策略
`policy`命令可以添加、删除、列出MinIO服务器上的策略。

```sh
NAME:
  mc admin policy - manage policies

FLAGS:
  --help, -h                       show help

COMMANDS:
  add      add new policy
  remove   remove policy
  list     List all policies
```

*示例： ‘使用/tmp/newpolicy.json上的策略在MinIO上添加一个新的策略‘newpolicy’。*

```sh
mc admin policy add myminio/ newpolicy /tmp/newpolicy.json
```

*示例： 移除在MinIO上的策略'newpolicy'*

```sh
mc admin policy remove myminio/ newpolicy
```

*示例： 列出MinIO上所有的策略。*

```sh
mc admin policy list --json myminio/
{"status":"success","policy":"newpolicy"}
```

<a name="user"></a>
### `user`命令 - 管理用户
`user`命令可以添加，删除，启用，禁用，列举minio服务器上的用户。

```sh
NAME:
  mc admin user - manage users

FLAGS:
  --help, -h                       show help

COMMANDS:
  add      add new user
  policy   set policy for user
  disable  disable user
  enable   enable user
  remove   remove user
  list     list all users
```

*示例： 用'newpolicy'策略在MinIO上添加一个新的用户'newuser'。*

```sh
mc admin user add myminio/ newuser newuser123 newpolicy
```

*示例：将MinIO上的用户'newuser'的策略更改为“writeonly'策略。*

```sh
mc admin user policy myminio/ newuser writeonly
```

*示例：在MinIO上禁用用户'newuser'*

```sh
mc admin user disable myminio/ newuser
```

*示例：在MinIO上启用用户'newuser'。*

```sh
mc admin user enable myminio/ newuser
```

*示例： 在MinIO上删除用户'newuser.'*

```sh
mc admin user remove myminio/ newuser
```

*示例： 在MinIO上列举所有的用户。*

```sh
mc admin user list --json myminio/
{"status":"success","accessKey":"newuser","userStatus":"enabled"}
```

<a name="config"></a>
### `config`命令 - 管理服务器配置

`config`命令可以管理MinIO服务器配置。

```sh
NAME:
  mc admin config - manage configuration file

USAGE:
  mc admin config COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  get  get config of a MinIO server/cluster.
  set  set new config file to a MinIO server/cluster.

FLAGS:
  --help, -h                       Show help.
```

*示例： 获取MinIO服务器/集群的服务器信息*

```sh
mc admin config get myminio > /tmp/my-serverconfig
```

*示例： 设置MinIO服务器/集群的服务器信息*

```sh
mc admin config set myminio < /tmp/my-serverconfig
```

<a name="heal"></a>

### `heal`命令 - 恢复MinIO上的磁盘，存储桶和对象

`heal`命令可以恢复在MinIO服务器上磁盘、丢失的存储桶以及对象。注意：这个命令仅仅适用于MinIO纠删码设置(单点或分布式)。


```sh
NAME:
  mc admin heal - heal disks, buckets and objects on MinIO server

FLAGS:
  --recursive, -r                  heal recursively
  --dry-run, -n                    only inspect data, but do not mutate
  --force-start, -f                force start a new heal sequence
  --help, -h                       show help
```

*示例：更换新磁盘后修复MinIO集群，递归修复所有的存储桶和对象，其中MinIO服务器别名为‘myminio’。*

```sh
mc admin heal -r myminio
```


*示例：递归地修复特定存储桶上的MinIO群集，其中MinIO服务器别名为‘myminio’。*

```sh
mc admin heal -r myminio/mybucket
```

*示例: 递归地修复具有一个特定对象前缀的MinIO集群，其中MinIO服务器的别名是‘myminio’*

```sh
mc admin heal -r myminio/mybucket/myobjectprefix
```

<a name="top"></a>

### `top`命令 - 为MinIO服务器提供最相似的统计数据
注意： 这个命令仅仅适用于分布式MinIO的设置。单点和网关部署都不支持该命令。

```
NAME:
  mc admin top - provide top like statistics for MinIO

FLAGS:
  --help, -h                    show help

COMMANDS:
  locks  Get a list of the 10 oldest locks on a MinIO cluster.
  
```

*例子：获取分布式MinIO集群上的时间最久的10把锁，其中MinIO分布式集群的别名是`myminio`。*

```sh
mc admin top locks myminio
```