#### list objects

```
NAME:
   mc ls - list files and objects

USAGE:
   mc ls TARGET

DESCRIPTION:
   List files and objects recursively between Amazon S3, Minio Object Storage and Filesystem

EXAMPLES:
   1. List objects on Minio object storage
      $ mc ls http://localhost:9000/backup/
      2015-03-28 12:47:50 PDT      51.00 MB 2006-Jan-1/backup.tar.gz
      2015-03-31 14:46:33 PDT      55.00 MB 2006-Mar-1/backup.tar.gz

   2. List buckets on Amazon S3 object storage
      $ mc ls https://s3.amazonaws.com/
      2015-01-20 15:42:00 PST               rom
      2015-01-15 00:05:40 PST               zek

```