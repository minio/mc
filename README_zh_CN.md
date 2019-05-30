# MinIO客户端快速入门指南
[![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io) [![Go Report Card](https://goreportcard.com/badge/minio/mc)](https://goreportcard.com/report/minio/mc) [![Docker Pulls](https://img.shields.io/docker/pulls/minio/mc.svg?maxAge=604800)](https://hub.docker.com/r/minio/mc/)

MinIO Client (mc)为ls，cat，cp，mirror，diff，find等UNIX命令提供了一种替代方案。它支持文件系统和兼容Amazon S3的云存储服务（AWS Signature v2和v4）。

```
ls       列出文件和文件夹。
mb       创建一个存储桶或一个文件夹
rb       移除存储桶
cat      显示文件和对象内容
head     显示对象的第一个'n'行
pipe     将一个STDIN重定向到一个对象或者文件或者STDOUT
share    生成用于共享的URL
cp       拷贝文件和对象
mirror   给存储桶和文件夹做镜像
find     查找文件
sql      对对象执行sql语句
stat     对象的统计内容
diff     列出两个文件夹或者存储桶的差异，如名字、大小和日期
rm       删除文件和对象
event    管理对象通知
watch    监听文件和对象的事件
policy   管理匿名对象访问
admin    管理minio服务器
session  为cp命令管理保存的会话
config   管理mc配置文件
update   检查软件更新
version  输出版本信息
```

## Docker容器
### 稳定版
```
docker pull minio/mc
docker run minio/mc ls play
```

### 尝鲜版
```
docker pull minio/mc:edge
docker run minio/mc:edge ls play
```

**注意:** 上述示例默认使用MinIO[演示环境](#test-your-setup)运行`mc`，如果想用`mc`操作其它S3兼容的服务，采用下面的方式来启动容器：

```sh
docker run -it --entrypoint=/bin/sh minio/mc
```

然后使用[`mc config`命令](#add-a-cloud-storage-service)。

## macOS
### Homebrew
使用[Homebrew](http://brew.sh/)安装mc。

```sh
brew install minio/stable/mc
mc --help
```

## GNU/Linux
### 下载二进制文件
| 平台 | CPU架构 | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.min.io/client/mc/release/linux-amd64/mc |
||64-bit PPC|https://dl.min.io/client/mc/release/linux-ppc64le/mc |

```sh
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc
./mc --help
```

## Microsoft Windows
### 下载二进制文件
| 平台 | CPU架构 | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit Intel|https://dl.min.io/client/mc/release/windows-amd64/mc.exe |

```sh
mc.exe --help
```

## 通过源码安装
通过源码安装仅适用于开发人员和高级用户。`mc update`命令不支持基于源码安装的更新通知。请从[https://min.io/download/#minio-client](https://min.io/download/#minio-client)下载官方版本。

如果您没有可工作的Golang环境，请参照[如何安装Golang](https://docs.min.io/cn/how-to-install-golang)。

```sh
go get -d github.com/minio/mc
cd ${GOPATH}/src/github.com/minio/mc
make
```

## 添加一个云存储服务
如果你打算仅在POSIX兼容文件系统中使用`mc`,那你可以直接略过本节，跳到[日常使用](#everyday-use)。

添加一个或多个S3亚马逊兼容的服务，请参考下面说明。`mc`将所有的配置信息都存储在``~/.mc/config.json``文件中。

```sh
mc config host add <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> --api <API-SIGNATURE> --lookup <BUCKET-LOOKUP-TYPE>
```

别名只是你云存储服务的简称。VS3终端（endpoint），访问密钥（access key）和密钥（secret key）是由你的云存储提供商提供的。API签名是可选参数，默认情况下，它被设置为"S3v4"。

Lookup是一个可选参数。它被用于指示dns和path样式的url请求是否被服务器支持，它接受“dns”，“path”和"auto"作为有效值。在默认情况下，它被设为“auto”，而且SDK会自动地决定所使用的url查找类型。

### 示例 - MinIO云存储
从MinIO服务获得URL、访问密钥（access key）和密钥（secret key）。

```sh
mc config host add minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
```

### 示例 - Amazon S3云存储
参考[AWS Credentials指南](https://docs.aws.amazon.com/zh_cn/general/latest/gr/aws-security-credentials.html)获取你的AccessKeyID和SecretAccessKey。

```sh
mc config host add s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 
```

**注意**：作为Amazon S3上的一个IAM使用者，你需要确保用户对存储桶的完全访问或为你的IAM用户设置下列限定的策略

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

### 示例 - Google云存储
参考[Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys)获取你的AccessKeyID和SecretAccessKey。

```sh
mc config host add gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 
```

注意：Google云存储只支持旧版签名版本V2，所以你需要选择S3v2。

## 验证你的设置
`mc`预先配置了云存储服务[https://play.min.io:9000](https://play.min.io:9000) ，别名“play”。它是一个用于研发和测试的MinIO服务。如果想测试Amazon S3,你可以用“s3”或者你在开始设置的别名来替换“play”。

*示例:*

列出[https://play.min.io:9000](https://play.min.io:9000)上的所有存储桶。

```sh
mc ls play
[2016-03-22 19:47:48 PDT]     0B my-bucketname/
[2016-03-22 22:01:07 PDT]     0B mytestbucket/
[2016-03-22 20:04:39 PDT]     0B mybucketname/
[2016-01-28 17:23:11 PST]     0B newbucket/
[2016-03-20 09:08:36 PDT]     0B s3git-test/
```

创建一个存储桶

`mb`命令用于创建一个新的存储桶。

*示例：*
```
mc mb play/mybucket
Bucket created succesfully 'play/mybucket'.
```

复制对象
`cp`命令用于从一个或多个源复制数据到目标文件。

*示例：*
```sh
mc cp myobject.txt play/mybucket
myobject.txt:    14 B / 14 B  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  100.00 % 41 B/s 0
```


<a name="everyday-use"></a>
## 日常使用

### Shell别名
你可以添加shell别名来覆盖默认的Unix工具命令。

```sh
alias ls='mc ls'
alias cp='mc cp'
alias cat='mc cat'
alias mkdir='mc mb'
alias pipe='mc pipe'
alias find='mc find'
```

### Shell自动补全
如果你正在使用bash或者zsh，当前的shell将会自动地设置为使用补全特性来为你的mc命令进行建议或自动补全。

```sh
mc <TAB>
admin    config   diff     find     ls       mirror   policy   session  sql      update   watch
cat      cp       event    head     mb       pipe     rm       share    stat     version
```

## 了解更多
- [MinIO Client完全指南](https://docs.min.io/cn/minio-client-complete-guide)
- [MinIO快速入门](https://docs.min.io/cn/minio-quickstart-guide)
- [MinIO官方文档](https://docs.min.io/cn)

## 为MinIO项目做贡献
请参照MinIO[贡献指南](https://github.com/minio/mc/blob/master/docs/zh_CN/CONTRIBUTING.md)

## 认证
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fminio%2Fmc.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fminio%2Fmc?ref=badge_large)
