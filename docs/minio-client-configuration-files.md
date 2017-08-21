# Minio Client Configuration Files Guide [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

In this document we will walk you through the configuration files of Minio Client.

## Minio Client configuration directory
Minio Client configurations are stored in file name ``.mc``.  It is a hidden file which resides on user's home directory.

**This how the structure of the directory looks like:**

```sh
tree ~/.mc
/home/supernova/.mc
├── config.json
├── session
└── share
2 directories, 5 files
```
### Files and directories

#### ``session`` directory
``session`` directory keeps metadata information of all incomplete upload or mirror. You can run ``mc session list`` to list the same. 

#### ``config.json``
config.json is the configuration file for Minio Client, it gets generated after you install and start Minio. All the credentials, endpoint information we add via ``mc config host`` are stored/modified here. 

```sh
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
			"url": "https://play.minio.io:9000",
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

``version`` tells the version of the file.

``hosts``  stores authentication credentials which will be used by Minio Client.

#### ``config.json.old``
This file keeps previous config file version details.

#### ``share`` directory
``share`` directory keeps metadata information of all upload and download URL for objects which is used by  Minio client ``mc share`` command. 

## Explore Further
* [Minio Client Complete Guide](https://docs.minio.io/docs/minio-client-complete-guide)




