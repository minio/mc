#### ls

```go
NAME:
   mc ls - List files and objects

USAGE:
   mc ls TARGET [TARGET...]

EXAMPLES:
   1. List objects recursively on Minio object storage
         $ mc ls http://play.minio.io:9000/backup/...
         [2015-03-28 12:47:50 PDT]  34MiB 2006-Jan-1/backup.tar.gz
         [2015-03-31 14:46:33 PDT]  55MiB 2006-Mar-1/backup.tar.gz

   2. List buckets on Amazon S3 object storage
         $ mc ls https://s3.amazonaws.com/
         [2015-01-20 15:42:00 PST]     0B rom/
         [2015-01-15 00:05:40 PST]     0B zek/

   3. List buckets from Amazon S3 object storage and recursively list objects from Minio object storage
         $ mc ls https://s3.amazonaws.com/ http://play.minio.io:9000/backup/...
	 [2015-01-15 00:05:40 PST]     0B zek/
	 [2015-03-31 14:46:33 PDT]  55MiB 2006-Mar-1/backup.tar.gz

   4. List files recursively on local filesystem on Windows
         $ mc ls C:\Users\Worf\...
         [2015-03-28 12:47:50 PDT]  11MiB Martok\Klingon Council Ministers.pdf
         [2015-03-31 14:46:33 PDT]  15MiB Gowron\Khitomer Conference Details.pdf

   5. List files non recursively on local filesystem
         $ mc ls  /usr/lib/llvm-3.4
         [2015-04-01 14:57:17 PDT]  12KiB lib/
         [2015-04-01 14:57:17 PDT] 4.0KiB include/
         [2015-04-01 14:57:10 PDT] 4.0KiB build/
         [2015-04-01 14:57:07 PDT] 4.0KiB bin/

```
