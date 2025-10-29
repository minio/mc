# MinIO客户端快速入门指南
[![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io) [![Go Report Card](https://goreportcard.com/badge/minio/mc)](https://goreportcard.com/report/minio/mc) [![Docker Pulls](https://img.shields.io/docker/pulls/minio/mc.svg?maxAge=604800)](https://hub.docker.com/r/minio/mc/)

MinIO Client (mc)为ls，cat，cp，mirror，diff，find等UNIX命令提供了一种替代方案。它支持文件系统和兼容Amazon S3的云存储服务（AWS Signature v2和v4）。


```
ls        列出文件和文件夹。
mb        创建一个存储桶或一个文件夹。
cat       显示文件和对象内容。
pipe      将一个STDIN重定向到一个对象或者文件或者STDOUT。
share     生成用于共享的URL。
cp        拷贝文件和对象。
mirror    给存储桶和文件夹做镜像。
find      基于参数查找文件。
diff      对两个文件夹或者存储桶比较差异。
rm        删除文件和对象。
events    管理对象通知。
watch     监听文件和对象的事件。
anonymous 管理访问策略。
session   为cp命令管理保存的会话。
config    管理mc配置文件。
update    检查软件更新。
version   输出版本信息。
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

**注意:** 上述示例默认使用MinIO[演示环境](#test-your-setup)做演示，如果想用`mc`操作其它S3兼容的服务，采用下面的方式来启动容器：

```
docker run -it --entrypoint=/bin/sh minio/mc
```

然后使用[`mc config`命令](#add-a-cloud-storage-service)。

## macOS
### Homebrew
使用[Homebrew](http://brew.sh/)安装mc。

```
brew install minio/stable/mc
mc --help
```

## GNU/Linux
### 下载二进制文件
| 平台 | CPU架构 | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.min.io/client/mc/release/linux-amd64/mc |

```
chmod +x mc
./mc --help
```

## Microsoft Windows
### 下载二进制文件
| 平台 | CPU架构 | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit Intel|https://dl.min.io/client/mc/release/windows-amd64/mc.exe |

```
mc.exe --help
```

## 通过源码安装
通过源码安装仅适用于开发人员和高级用户。`mc update`命令不支持基于源码安装的更新通知。请从https://min.io/download/#minio-client下载官方版本。

如果您没有Golang环境，请参照[如何安装Golang](https://golang.org/doc/install)。

```
go get -d github.com/minio/mc
cd ${GOPATH}/src/github.com/minio/mc
make
```

## 添加一个云存储服务
如果你打算仅在POSIX兼容文件系统中使用`mc`,那你可以直接略过本节，跳到[日常使用](#everyday-use)。

添加一个或多个S3兼容的服务，请参考下面说明。`mc`将所有的配置信息都存储在``~/.mc/config.json``文件中。

```
mc alias set <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> [--api API-SIGNATURE]
```

别名就是给你的云存储服务起了一个短点的外号。S3 endpoint,access key和secret key是你的云存储服务提供的。API签名是可选参数，默认情况下，它被设置为"S3v4"。

### 示例-MinIO云存储
从MinIO服务获得URL、access key和secret key。

```
mc alias set minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 --api s3v4
```

### 示例-Amazon S3云存储
参考[AWS Credentials指南](http://docs.aws.amazon.com/general/latest/gr/aws-security-credentials.html)获取你的AccessKeyID和SecretAccessKey。

```
mc alias set s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 --api s3v4
```

### 示例-Google云存储
参考[Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys)获取你的AccessKeyID和SecretAccessKey。

```
mc alias set gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 --api s3v2
```

注意：Google云存储只支持旧版签名版本V2，所以你需要选择S3v2。

## 验证
`mc`预先配置了云存储服务URL：https://play.min.io，别名“play”。它是一个用于研发和测试的MinIO服务。如果想测试Amazon S3,你可以将“play”替换为“s3”。

*示例:*

列出https://play.min.io上的所有存储桶。

```
mc ls play
[2016-03-22 19:47:48 PDT]     0B my-bucketname/
[2016-03-22 22:01:07 PDT]     0B mytestbucket/
[2016-03-22 20:04:39 PDT]     0B mybucketname/
[2016-01-28 17:23:11 PST]     0B newbucket/
[2016-03-20 09:08:36 PDT]     0B s3git-test/
```
<a name="everyday-use"></a>
## 日常使用

### Shell别名
你可以添加shell别名来覆盖默认的Unix工具命令。

```
alias ls='mc ls'
alias cp='mc cp'
alias cat='mc cat'
alias mkdir='mc mb'
alias pipe='mc pipe'
alias find='mc find'
```

### Shell自动补全
你也可以下载[`autocomplete/bash_autocomplete`](https://raw.githubusercontent.com/minio/mc/master/autocomplete/bash_autocomplete)到`/etc/bash_completion.d/`，然后将其重命名为`mc`。别忘了在这个文件运行source命令让其在你的当前shell上可用。

```
sudo wget https://raw.githubusercontent.com/minio/mc/master/autocomplete/bash_autocomplete -O /etc/bash_completion.d/mc
source /etc/bash_completion.d/mc
```

```
mc <TAB>
admin    config   diff     ls       mirror   policy   session  update   watch
cat      cp       events   mb       pipe     rm       share    version
```

## 了解更多
- [MinIO Client完全指南](https://docs.min.io/community/minio-object-store/reference/minio-mc.html?ref=gh)
- [MinIO快速入门](https://docs.min.io/community/minio-object-store/operations/deployments/baremetal-deploy-minio-on-redhat-linux.html?ref=gh)
- [MinIO官方文档](https://docs.min.io/community/minio-object-store/index.html?ref=gh)

## 贡献
请遵守MinIO[贡献者指南](https://github.com/minio/mc/blob/master/docs/zh_CN/CONTRIBUTING.md)
