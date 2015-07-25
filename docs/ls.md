#### ls

```go
NAME:
   mc ls - List files and folders

USAGE:
   mc ls TARGET [TARGET...]

EXAMPLES:
   1. List objects recursively on Minio cloud storage.
      $ mc ls https://play.minio.io:9000/backup/...
      [2015-03-28 12:47:50 PDT]  34MiB 2006-Jan-1/backup.tar.gz
      [2015-03-31 14:46:33 PDT]  55MiB 2006-Mar-1/backup.tar.gz

   2. List buckets on Amazon S3 cloud storage.
      $ mc ls https://s3.amazonaws.com/
      [2015-01-20 15:42:00 PST]     0B rom/
      [2015-01-15 00:05:40 PST]     0B zek/

   3. List buckets from Amazon S3 cloud storage and recursively list objects from Minio cloud storage.
      $ mc ls https://s3.amazonaws.com/ https://play.minio.io:9000/backup/...
      2015-01-15 00:05:40 PST     0B zek/
      2015-03-31 14:46:33 PDT  55MiB 2006-Mar-1/backup.tar.gz

   4. List files recursively on local filesystem on Windows.
      $ mc ls C:\Users\Worf\...
      [2015-03-28 12:47:50 PDT] 11.00MiB Martok\Klingon Council Ministers.pdf
      [2015-03-31 14:46:33 PDT] 15.00MiB Gowron\Khitomer Conference Details.pdf

   5. List files with non english characters on Amazon S3 cloud storage.
      $ mc ls s3:andoria/本...
      [2015-05-19 17:21:49 PDT]    41B 本語.pdf
      [2015-05-19 17:24:19 PDT]    41B 本語.txt
      [2015-05-19 17:28:22 PDT]    41B 本語.md

```
