# MinIO Client配置文件指南 [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

本文我们将详细介绍MinIO Client的配置文件。

## MinIO Client配置目录
MinIO Client配置信息存储在``.mc``文件夹，它是用户home目录下的一个隐藏文件夹。

**这就是配置文件夹的目录结构：**

```
tree ~/.mc
/home/supernova/.mc
├── config.json
├── session
└── share
2 directories, 5 files
```
### 文件和目录

#### ``session``目录
``session``目录保存所有不完整上传或镜像的元数据信息。你可以运行`mc session list``列出这些信息。

#### ``config.json``
config.json是MinIO Client的配置文件，它在安装并启动MinIO后生成。我们通过``mc config host``添加的所有凭证，endpoint信息都存储在这里。

```
cat config.json 
{
	"version": "8",
	"hosts": {
		"XL": {
			"url": "http://127.0.0.1:9000",
			"accessKey": "YI7S1CKXB76RGOGT6R8W",
			"secretKey": "FJ9PWUVNXGPfiI72WMRFepN3LsFgW3MjsxSALroV",
			"api": "S3v4"
		},
		"fs": {
			"url": "http://127.0.0.1:9000",
			"accessKey": "YI7S1CKXB76RGOGT6R8W",
			"secretKey": "FJ9PWUVNXGPfiI72WMRFepN3LsFgW3MjsxSALroV",
			"api": "S3v4"
		},
		"gcs": {
			"url": "https://storage.googleapis.com",
			"accessKey": "YOUR-ACCESS-KEY-HERE",
			"secretKey": "YOUR-SECRET-KEY-HERE",
			"api": "S3v2"
		},
		"play": {
			"url": "https://play.min.io",
			"accessKey": "Q3AM3UQ867SPQQA43P2F",
			"secretKey": "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
			"api": "S3v4"
		},
		"s3": {
			"url": "https://s3.amazonaws.com",
			"accessKey": "YOUR-ACCESS-KEY-HERE",
			"secretKey": "YOUR-SECRET-KEY-HERE",
			"api": "S3v4"
		}
	}
}
```

``version``代表的是这个文件的版本。

``hosts``存储将被MinIO Client使用的认证证书。

#### ``config.json.old``
这个文件保存了以前的配置文件版本细节。

#### ``share``目录
``share``目录保存MinIO Client ``mc share``命令使用的所有对象的上传和下载URL的元数据信息。

## 了解更多
* [MinIO Client完全指南](https://docs.min.io/docs/minio-client-complete-guide)




