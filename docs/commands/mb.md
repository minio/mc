#### mb

```go
NAME:
   mc mb - Make a bucket or a folder

USAGE:
   mc mb TARGET [TARGET...]

EXAMPLES:
   1. Create a bucket on Amazon S3 object storage
         $ mc mb https://s3.amazonaws.com/public-document-store

   2. Create a bucket on Minio object storage
         $ mc mb https://play.minio.io:9000/mongodb-backup

   3. Create multiple buckets on Amazon S3 object storage and Minio object storage
         $ mc mb https://s3.amazonaws.com/public-photo-store https://play.minio.io:9000/mongodb-backup
```
