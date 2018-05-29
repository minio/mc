# Minio Client完全指南 [![Slack](https://slack.minio.io/slack?type=svg)](https://slack.minio.io)

Minio Client (mc)为ls，cat，cp，mirror，diff，find等UNIX命令提供了一种替代方案。它支持文件系统和兼容Amazon S3的云存储服务（AWS Signature v2和v4）。

```sh
ls       列出文件和文件夹。
mb       创建一个存储桶或一个文件夹。
cat      显示文件和对象内容。
pipe     将一个STDIN重定向到一个对象或者文件或者STDOUT。
share    生成用于共享的URL。
cp       拷贝文件和对象。
mirror   给存储桶和文件夹做镜像。
find     基于参数查找文件。
diff     对两个文件夹或者存储桶比较差异。
rm       删除文件和对象。
events   管理对象通知。
watch    监视文件和对象的事件。
policy   管理访问策略。
session  为cp命令管理保存的会话。
config   管理mc配置文件。
update   检查软件更新。
version  输出版本信息。
```

## 1.  下载Minio Client
### Docker稳定版
```
docker pull minio/mc
docker run minio/mc ls play
```

### Docker尝鲜版
```
docker pull minio/mc:edge
docker run minio/mc:edge ls play
```

**注意:** 上述示例默认使用Minio[演示环境](#test-your-setup)做演示，如果想用`mc`操作其它S3兼容的服务，采用下面的方式来启动容器：

```sh
docker run -it --entrypoint=/bin/sh minio/mc
```

然后使用[`mc config`命令](#add-a-cloud-storage-service)。

### Homebrew (macOS)
使用[Homebrew](http://brew.sh/)安装mc。

```sh
brew install minio/stable/mc
mc --help
```

### 下载二进制文件(GNU/Linux)
| 平台 | CPU架构 | URL |
| ---------- | -------- |------|
|GNU/Linux|64-bit Intel|https://dl.minio.io/client/mc/release/linux-amd64/mc |

```sh
chmod +x mc
./mc --help
```

### 下载二进制文件(Microsoft Windows)
| 平台 | CPU架构 | URL |
| ---------- | -------- |------|
|Microsoft Windows|64-bit Intel|https://dl.minio.io/client/mc/release/windows-amd64/mc.exe |

```sh
mc.exe --help
```

### 通过源码安装
通过源码安装仅适用于开发人员和高级用户。`mc update`命令不支持基于源码安装的更新通知。请从[minio-client](https://minio.io/downloads/#minio-client)下载官方版本。

如果您没有Golang环境，请按照 [如何安装Golang](https://docs.minio.io/docs/how-to-install-golang)。

```sh
go get -d github.com/minio/mc
cd ${GOPATH}/src/github.com/minio/mc
make
```

## 2. 运行Minio Client

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

## 3. 添加一个云存储服务
如果你打算仅在POSIX兼容文件系统中使用`mc`,那你可以直接略过本节，跳到**Step 4**。

添加一个或多个S3兼容的服务，请参考下面说明。`mc`将所有的配置信息都存储在``~/.mc/config.json``文件中。

#### 使用

```sh
mc config host add <ALIAS> <YOUR-S3-ENDPOINT> <YOUR-ACCESS-KEY> <YOUR-SECRET-KEY> <API-SIGNATURE>
```

别名就是给你的云存储服务起了一个短点的外号。S3 endpoint,access key和secret key是你的云存储服务提供的。API签名是可选参数，默认情况下，它被设置为"S3v4"。

### 示例-Minio云存储
从Minio服务获得URL、access key和secret key。


```sh
mc config host add minio http://192.168.1.51 BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v4
```

### 示例-Amazon S3云存储
参考[AWS Credentials指南](http://docs.aws.amazon.com/general/latest/gr/aws-security-credentials.html)获取你的AccessKeyID和SecretAccessKey。

```sh
mc config host add s3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v4
```

### 示例-Google云存储
参考[Google Credentials Guide](https://cloud.google.com/storage/docs/migrating?hl=en#keys)获取你的AccessKeyID和SecretAccessKey。

```sh
mc config host add gcs  https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v2
```

注意：Google云存储只支持旧版签名版本V2，所以你需要选择S3v2。

## 4. 验证
`mc`预先配置了云存储服务URL：[https://play.minio.io:9000](https://play.minio.io:9000)，别名“play”。它是一个用于研发和测试的Minio服务。如果想测试Amazon S3,你可以将“play”替换为“s3”。

*示例:*

列出[https://play.minio.io:9000](https://play.minio.io:9000)上的所有存储桶。

```sh
mc ls play
[2016-03-22 19:47:48 PDT]     0B my-bucketname/
[2016-03-22 22:01:07 PDT]     0B mytestbucket/
[2016-03-22 20:04:39 PDT]     0B mybucketname/
[2016-01-28 17:23:11 PST]     0B newbucket/
[2016-03-20 09:08:36 PDT]     0B s3git-test/
```

## 5. 日常使用
你可以添加shell别名来覆盖默认的Unix工具命令。

```sh
alias ls='mc ls'
alias cp='mc cp'
alias cat='mc cat'
alias mkdir='mc mb'
alias pipe='mc pipe'
alias find='mc find'
```

## 6. 全局参数

### 参数 [--debug]
Debug参数开启控制台输出debug信息。

*示例：输出`ls`命令的详细debug信息。*

```sh
mc --debug ls play
mc: <DEBUG> GET / HTTP/1.1
Host: play.minio.io:9000
User-Agent: Minio (darwin; amd64) minio-go/1.0.1 mc/2016-04-01T00:22:11Z
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
Server: Minio/DEVELOPMENT.2016-04-07T18-53-27Z (linux; amd64)
Vary: Origin
X-Amz-Request-Id: HP30I0W2U49BDBIO

mc: <DEBUG> Response Time:  1.220112837s

[...]

[2016-04-08 03:56:14 IST]     0B albums/
[2016-04-04 16:11:45 IST]     0B backup/
[2016-04-01 20:10:53 IST]     0B deebucket/
[2016-03-28 21:53:49 IST]     0B guestbucket/
```

### 参数 [--json]
JSON参数启用JSON格式的输出。

*示例：列出Minio play服务的所有存储桶。*

```sh
mc --json ls play
{"status":"success","type":"folder","lastModified":"2016-04-08T03:56:14.577+05:30","size":0,"key":"albums/"}
{"status":"success","type":"folder","lastModified":"2016-04-04T16:11:45.349+05:30","size":0,"key":"backup/"}
{"status":"success","type":"folder","lastModified":"2016-04-01T20:10:53.941+05:30","size":0,"key":"deebucket/"}
{"status":"success","type":"folder","lastModified":"2016-03-28T21:53:49.217+05:30","size":0,"key":"guestbucket/"}
```

### 参数 [--no-color]
这个参数禁用颜色主题。对于一些比较老的终端有用。

### 参数 [--quiet]
这个参数关闭控制台日志输出。

### 参数 [--config-folder]
这个参数参数自定义的配置文件路径。

### 参数 [ --insecure]
跳过SSL证书验证。

## 7. 命令

|   |   | |
|:---|:---|:---|
|[**ls** - 列出存储桶和对象](#ls)   |[**mb** - 创建存储桶](#mb)  | [**cat** - 合并对象](#cat)  |
|[**cp** - 拷贝对象](#cp) | [**rm** - 删除对象](#rm)  | [**pipe** - Pipe到一个对象](#pipe) |
| [**share** - 共享](#share)  |[**mirror** - 存储桶镜像](#mirror)  | [**find** - 查找文件和对象](#find) |
| [**diff** - 比较存储桶差异](#diff) |[**policy** - 给存储桶或前缀设置访问策略](#policy)  |[**session** - 管理保存的会话](#session) |
| [**config** - 管理配置文件](#config)  | [**watch** - 事件监听](#watch)  | [**events** - 管理存储桶事件](#events)  |
| [**update** - 管理软件更新](#update)  | [**version** - 显示版本信息](#version)  |   |


###  `ls`命令 - 列出对象
`ls`命令列出文件、对象和存储桶。使用`--incomplete` flag可列出未完整拷贝的内容。

```sh
用法：
   mc ls [FLAGS] TARGET [TARGET ...]

FLAGS:
  --help, -h                       显示帮助。
  --recursive, -r		   递归。
  --incomplete, -I		   列出未完整上传的对象。
```

*示例： 列出所有https://play.minio.io:9000上的存储桶。*

```sh
mc ls play
[2016-04-08 03:56:14 IST]     0B albums/
[2016-04-04 16:11:45 IST]     0B backup/
[2016-04-01 20:10:53 IST]     0B deebucket/
[2016-03-28 21:53:49 IST]     0B guestbucket/
[2016-04-08 20:58:18 IST]     0B mybucket/
```
<a name="mb"></a>
### `mb`命令 - 创建存储桶
`mb`命令在对象存储上创建一个新的存储桶。在文件系统，它就和`mkdir -p`命令是一样的。存储桶相当于文件系统中的磁盘或挂载点，不应视为文件夹。Minio对每个​​用户创建的存储桶数量没有限制。
在Amazon S3上，每个帐户被限制为100个存储桶。有关更多信息，请参阅[S3上的存储桶限制和限制](http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html) 。

```sh
用法：
   mc mb [FLAGS] TARGET [TARGET...]

FLAGS:
  --help, -h                       显示帮助。
  --region "us-east-1"		   指定存储桶的region，默认是‘us-east-1’.

```

*示例：在https://play.minio.io:9000上创建一个名叫"mybucket"的存储桶。*


```sh
mc mb play/mybucket
Bucket created successfully ‘play/mybucket’.
```

<a name="cat"></a>

### `cat`命令 - 合并对象
`cat`命令将一个文件或者对象的内容合并到另一个上。你也可以用它将对象的内容输出到stdout。

```sh
用法：
   mc cat [FLAGS] SOURCE [SOURCE...]

FLAGS:
  --help, -h                       显示帮助。
```

*示例： 显示`myobject.txt`文件的内容*

```sh
mc cat play/mybucket/myobject.txt
Hello Minio!!
```
<a name="pipe"></a>
### `pipe`命令 - Pipe到对象
`pipe`命令拷贝stdin里的内容到目标输出，如果没有指定目标输出，则输出到stdout。

```sh
用法：
   mc pipe [FLAGS] [TARGET]

FLAGS:
  --help, -h					显示帮助。
```

*示例： 将MySQL数据库dump文件输出到Amazon S3。*

```sh
mysqldump -u root -p ******* accountsdb | mc pipe s3/ferenginar/backups/accountsdb-oct-9-2015.sql
```

<a name="cp"></a>
### `cp`命令 - 拷贝对象
`cp`命令拷贝一个或多个源文件目标输出。所有到对象存储的拷贝操作都进行了MD4SUM checkSUM校验。可以从故障点恢复中断或失败的复制操作。

```sh
用法：
   mc cp [FLAGS] SOURCE [SOURCE...] TARGET

FLAGS:
  --help, -h                       显示帮助。
  --recursive, -r		   递归拷贝。
```

*示例： 拷贝一个文本文件到对象存储。*

```sh
mc cp myobject.txt play/mybucket
myobject.txt:    14 B / 14 B  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  100.00 % 41 B/s 0
```
<a name="rm"></a>
### `rm`命令 - 删除存储桶和对象。
使用`rm`命令删除文件对象或者存储桶。

```sh
用法：
   mc rm [FLAGS] TARGET [TARGET ...]

FLAGS:
  --help, -h                       显示帮助。
  --recursive, -r	       	   递归删除。
  --force			   强制执行删除操作。
  --prefix			   删除批配这个前缀的对象。
  --incomplete, -I		删除未完整上传的对象。
  --fake			   模拟一个假的删除操作。
  --stdin			   从STDIN中读对象列表。
  --older-than value               删除N天前的对象（默认是0天）。
```

*示例： 删除一个对象。*

```sh
mc rm play/mybucket/myobject.txt
Removed ‘play/mybucket/myobject.txt’.
```

*示例：删除一个存储桶并递归删除里面所有的内容。由于这个操作太危险了，你必须传`--force`参数指定强制删除。*

```sh
mc rm --recursive --force play/myobject
Removed ‘play/myobject/newfile.txt’.
Removed 'play/myobject/otherobject.txt’.
```

*示例： 从`mybucket`里删除所有未完整上传的对象。*

```sh
mc rm  --incomplete --recursive --force play/mybucket
Removed ‘play/mybucket/mydvd.iso’.
Removed 'play/mybucket/backup.tgz’.
```
*示例： 删除一天前的对象。*

```sh
mc rm --force --older-than=1 play/mybucket/oldsongs
```

<a name="share"></a>
### `share`命令 - 共享
`share`命令安全地授予上传或下载的权限。此访问只是临时的，与远程用户和应用程序共享也是安全的。如果你想授予永久访问权限，你可以看看`mc policy`命令。

生成的网址中含有编码后的访问认证信息，任何企图篡改URL的行为都会使访问无效。想了解这种机制是如何工作的，请参考[Pre-Signed URL](http://docs.aws.amazon.com/AmazonS3/latest/dev/ShareObjectPreSignedURL.html)技术。

```sh
用法：
   mc share [FLAGS] COMMAND

FLAGS:
  --help, -h                       显示帮助。

COMMANDS:
   download	  生成有下载权限的URL。
   upload	  生成有上传权限的URL。
   list		  列出先前共享的对象和文件夹。
```

### 子命令`share download` - 共享下载
`share download`命令生成不需要access key和secret key即可下载的URL，过期参数设置成最大有效期（不大于7天），过期之后权限自动回收。

```sh
用法：
   mc share download [FLAGS] TARGET [TARGET...]

FLAGS:
  --help, -h                       显示帮助。
  --recursive, -r		   递归共享所有对象。
  --expire, -E "168h"		   设置过期时限，NN[h|m|s]。
```

*示例： 生成一个对一个对象有4小时访问权限的URL。*

```sh

mc share download --expire 4h play/mybucket/myobject.txt
URL: https://play.minio.io:9000/mybucket/myobject.txt
Expire: 0 days 4 hours 0 minutes 0 seconds
Share: https://play.minio.io:9000/mybucket/myobject.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=Q3AM3UQ867SPQQA43P2F%2F20160408%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20160408T182008Z&X-Amz-Expires=604800&X-Amz-SignedHeaders=host&X-Amz-Signature=1527fc8f21a3a7e39ce3c456907a10b389125047adc552bcd86630b9d459b634

```

#### 子命令`share upload` - 共享上传
`share upload`命令生成不需要access key和secret key即可上传的URL。过期参数设置成最大有效期（不大于7天），过期之后权限自动回收。
Content-type参数限制只允许上传指定类型的文件。

```sh
用法：
   mc share upload [FLAGS] TARGET [TARGET...]

FLAGS:
  --help, -h                       显示帮助。
  --recursive, -r   		   递归共享所有对象。
  --expire, -E "168h"		   设置过期时限，NN[h|m|s].
```

*示例： 生成一个`curl`命令，赋予上传到`play/mybucket/myotherobject.txt`的权限。*

```sh
mc share upload play/mybucket/myotherobject.txt
URL: https://play.minio.io:9000/mybucket/myotherobject.txt
Expire: 7 days 0 hours 0 minutes 0 seconds
Share: curl https://play.minio.io:9000/mybucket -F x-amz-date=20160408T182356Z -F x-amz-signature=de343934bd0ba38bda0903813b5738f23dde67b4065ea2ec2e4e52f6389e51e1 -F bucket=mybucket -F policy=eyJleHBpcmF0aW9uIjoiMjAxNi0wNC0xNVQxODoyMzo1NS4wMDdaIiwiY29uZGl0aW9ucyI6W1siZXEiLCIkYnVja2V0IiwibXlidWNrZXQiXSxbImVxIiwiJGtleSIsIm15b3RoZXJvYmplY3QudHh0Il0sWyJlcSIsIiR4LWFtei1kYXRlIiwiMjAxNjA0MDhUMTgyMzU2WiJdLFsiZXEiLCIkeC1hbXotYWxnb3JpdGhtIiwiQVdTNC1ITUFDLVNIQTI1NiJdLFsiZXEiLCIkeC1hbXotY3JlZGVudGlhbCIsIlEzQU0zVVE4NjdTUFFRQTQzUDJGLzIwMTYwNDA4L3VzLWVhc3QtMS9zMy9hd3M0X3JlcXVlc3QiXV19 -F x-amz-algorithm=AWS4-HMAC-SHA256 -F x-amz-credential=Q3AM3UQ867SPQQA43P2F/20160408/us-east-1/s3/aws4_request -F key=myotherobject.txt -F file=@<FILE>
```

#### 子命令`share list` - 列出之前的共享
`share list`列出没未过期的共享URL。

```sh
用法：
   mc share list COMMAND

COMMAND:
   upload:   列出先前共享的有上传权限的URL。
   download: 列出先前共享的有下载权限的URL。
```

<a name="mirror"></a>
### `mirror`命令 - 存储桶镜像
`mirror`命令和`rsync`类似，只不过它是在文件系统和对象存储之间做同步。

```sh
用法：
   mc mirror [FLAGS] SOURCE TARGET

FLAGS:
  --help, -h                       显示帮助。
  --force			   强制覆盖已经存在的目标。
  --fake			   模拟一个假的操作。
  --watch, -w                      监听改变并执行镜像操作。
  --remove			   删除目标上的外部的文件。
```

*示例： 将一个本地文件夹镜像到https://play.minio.io:9000上的'mybucket'存储桶。*

```sh
mc mirror localdir/ play/mybucket
localdir/b.txt:  40 B / 40 B  ┃▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓┃  100.00 % 73 B/s 0
```

*示例： 持续监听本地文件夹修改并镜像到https://play.minio.io:9000上的'mybucket'存储桶。*

```sh
mc mirror -w localdir play/mybucket
localdir/new.txt:  10 MB / 10 MB  ┃▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓┃  100.00 % 1 MB/s 15s
```

<a name="find"></a>
### `find`命令 - 查找文件和对象
``find``命令通过指定参数查找文件，它只列出满足条件的数据。

```sh
用法：
  mc find PATH [FLAGS]

FLAGS:
  --help, -h                       显示帮助。
  --exec value                     为每个匹配对象生成一个外部进程（请参阅FORMAT）
  --name value                     查找匹配通配符模式的对象。
  ...
  ...
```

*示例： 持续从s3存储桶中查找所有jpeg图像，并复制到minio "play/bucket"存储桶*
```sh
mc find s3/bucket --name "*.jpg" --watch --exec "mc cp {} play/bucket"
```

<a name="diff"></a>
### `diff`命令 - 显示差异
``diff``命令计算两个目录之间的差异。它只列出缺少的或者大小不同的内容。

它*不*比较内容，所以可能的是，名称相同，大小相同但内容不同的对象没有被检测到。这样，它可以在不同站点或者大量数据的情况下快速比较。

```sh
用法：
  mc diff [FLAGS] FIRST SECOND

FLAGS:
  --help, -h                       显示帮助。
```

*示例： 比较一个本地文件夹和一个远程对象存储服务*

```sh
 mc diff localdir play/mybucket
‘localdir/notes.txt’ and ‘https://play.minio.io:9000/mybucket/notes.txt’ - only in first.
```

<a name="watch"></a>
### `watch`命令 - 监听文件和对象存储事件。
``watch``命令提供了一种方便监听对象存储和文件系统上不同类型事件的方式。

```sh
用法：
  mc watch [FLAGS] PATH

FLAGS:
  --events value                   过滤不同类型的事件，默认是所有类型的事件 (默认： "put,delete,get")
  --prefix value                   基于前缀过滤事件。
  --suffix value                   基于后缀过滤事件。
  --recursive                      递归方式监听事件。
  --help, -h                       显示帮助。
```

*示例： 监听对象存储的所有事件*

```sh
mc watch play/testbucket
[2016-08-18T00:51:29.735Z] 2.7KiB ObjectCreated https://play.minio.io:9000/testbucket/CONTRIBUTING.md
[2016-08-18T00:51:29.780Z]  1009B ObjectCreated https://play.minio.io:9000/testbucket/MAINTAINERS.md
[2016-08-18T00:51:29.839Z] 6.9KiB ObjectCreated https://play.minio.io:9000/testbucket/README.md
```

*示例： 监听本地文件夹的所有事件*

```sh
mc watch ~/Photos
[2016-08-17T17:54:19.565Z] 3.7MiB ObjectCreated /home/minio/Downloads/tmp/5467026530_a8611b53f9_o.jpg
[2016-08-17T17:54:19.565Z] 3.7MiB ObjectCreated /home/minio/Downloads/tmp/5467026530_a8611b53f9_o.jpg
...
[2016-08-17T17:54:19.565Z] 7.5MiB ObjectCreated /home/minio/Downloads/tmp/8771468997_89b762d104_o.jpg
```

<a name="events"></a>
### `events`命令 - 管理存储桶事件通知。
``events``提供了一种方便的配置存储桶的各种类型事件通知的方式。Minio事件通知可以配置成使用 AMQP，Redis，ElasticSearch，NATS和PostgreSQL服务。Minio configuration提供了如何配置的更多细节。

```sh
用法：
  mc events COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  add     添加一个新的存储桶通知。
  remove  删除一个存储桶通知。使用'--force'可以删除所有存储桶通知。
  list    列出存储桶通知。

FLAGS:
  --help, -h                       显示帮助。
```

*示例： 列出所有存储桶通知。*

```sh
mc events list play/andoria
MyTopic        arn:minio:sns:us-east-1:1:TestTopic    s3:ObjectCreated:*,s3:ObjectRemoved:*   suffix:.jpg
```

*示例： 添加一个新的'sqs'通知，仅接收ObjectCreated事件。*

```sh
mc events add play/andoria arn:minio:sqs:us-east-1:1:your-queue --events put
```

*示例： 添加一个带有过滤器的'sqs'通知。*

给`sqs`通知添加`prefix`和`suffix`过滤规则。

```sh
mc events add play/andoria arn:minio:sqs:us-east-1:1:your-queue --prefix photos/ --suffix .jpg
```

*示例： 删除一个'sqs'通知*

```sh
mc events remove play/andoria arn:minio:sqs:us-east-1:1:your-queue
```

<a name="policy"></a>
### `policy`命令 - 管理存储桶策略
管理匿名访问存储桶和其内部内容的策略。

```sh
用法：
  mc policy [FLAGS] PERMISSION TARGET
  mc policy [FLAGS] TARGET
  mc policy list [FLAGS] TARGET

PERMISSION:
  Allowed policies are: [none, download, upload, public].

FLAGS:
  --help, -h                       显示帮助。
```

*示例： 显示当前匿名存储桶策略*

显示当前``mybucket/myphotos/2020/``子文件夹的匿名策略。

```sh
mc policy play/mybucket/myphotos/2020/
Access permission for ‘play/mybucket/myphotos/2020/’ is ‘none’
```

*示例：设置可下载的匿名存储桶策略。*

设置``mybucket/myphotos/2020/``子文件夹可匿名下载的策略。现在，这个文件夹下的对象可被公开访问。比如：``mybucket/myphotos/2020/yourobjectname``可通过这个URL [https://play.minio.io:9000/mybucket/myphotos/2020/yourobjectname](https://play.minio.io:9000/mybucket/myphotos/2020/yourobjectname)访问。

```sh
mc policy download play/mybucket/myphotos/2020/
Access permission for ‘play/mybucket/myphotos/2020/’ is set to 'download'
```

*示例：删除当前的匿名存储桶策略*

删除所有*mybucket/myphotos/2020/*这个子文件夹下的匿名存储桶策略。

```sh
mc policy none play/mybucket/myphotos/2020/
Access permission for ‘play/mybucket/myphotos/2020/’ is set to 'none'
```

<a name="session"></a>
### `session`命令 - 管理session
``session``命令管理之前保存的`cp`和`mirror`操作的会话。

```sh
用法：
  mc session COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  list    列出所有之前保存的会话。
  clear   清除某个之前保存的会话。
  resume  恢复某个之前保存的会话。

FLAGS:
  --help, -h                       显示帮助。

```

*示例： 列出所有之前保存的会话*

```sh
mc session list
IXWKjpQM -> [2016-04-08 19:11:14 IST] cp assets.go play/mybucket
ApwAxSwa -> [2016-04-08 01:49:19 IST] mirror miniodoc/ play/mybucket
```

*示例： 恢复一个之前保存的会话*

```sh
mc session resume IXWKjpQM
...assets.go: 1.68 KB / 1.68 KB  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  100.00 % 784 B/s 2s
```

*示例： 清除一个之前保存的会话。*

```sh
mc session clear ApwAxSwa
Session ‘ApwAxSwa’ cleared successfully.
```

<a name="config"></a>
### `config`命令 - 管理配置文件
`config host`命令提供了一个方便地管理`~/.mc/config.json`配置文件中的主机信息的方式，你也可以用文本编辑器手动修改这个配置文件。

```sh
用法：
  mc config host COMMAND [COMMAND FLAGS | -h] [ARGUMENTS...]

COMMANDS:
  add, a      添加一个新的主机到配置文件。
  remove, rm  从配置文件中删除一个主机。
  list, ls    列出配置文件中的主机。

FLAGS:
  --help, -h                       显示帮助。
```

*示例： 管理配置文件*

添加Minio服务的access和secret key到配置文件，注意，shell的history特性可能会记录这些信息，从而带来安全隐患。在`bash` shell,使用`set -o`和`set +o`来关闭和开启history特性。

```sh
set +o history
mc config host add myminio http://localhost:9000 OMQAGGOL63D7UNVQFY8X GcY5RHNmnEWvD/1QxD3spEIGj+Vt9L7eHaAaBTkJ
set -o history
```

<a name="update"></a>
### `update`命令 - 软件更新
从[https://dl.minio.io](https://dl.minio.io)检查软件更新。Experimental标志会检查unstable实验性的版本，通常用作测试用途。

```sh
用法：
  mc update [FLAGS]

FLAGS:
  --quiet, -q  关闭控制台输出。
  --json       使用JSON格式输出。
  --help, -h   显示帮助。
```

*示例： 检查更新*

```sh
mc update
You are already running the most recent version of ‘mc’.
```

<a name="version"></a>
### `version`命令 - 显示版本信息
显示当前安装的`mc`版本。

```sh
用法：
  mc version [FLAGS]

FLAGS:
  --quiet, -q  关闭控制台输出。
  --json       使用JSON格式输出。
  --help, -h   显示帮助。
```

 *示例： 输出mc版本。*

```sh
mc version
Version: 2016-04-01T00:22:11Z
Release-tag: RELEASE.2016-04-01T00-22-11Z
Commit-id: 12adf3be326f5b6610cdd1438f72dfd861597fce
```
