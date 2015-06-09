#### cp

```go
NAME:
   mc cp - Copy objects and files from multiple sources single destination

USAGE:
   mc cp SOURCE TARGET [TARGET...]

EXAMPLES:
   1. Copy list of objects from local file system to Amazon S3 object storage
         $ mc cp Music/*.ogg https://s3.amazonaws.com/jukebox/

   2. Copy a bucket recursively from Minio object storage to Amazon S3 object storage
         $ mc cp https://play.minio.io:9000/photos/burningman2011... https://s3.amazonaws.com/private-photos/burningman/

   3. Copy multiple local folders recursively to Minio object storage
         $ mc cp backup/2014/... backup/2015/... https://play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 object storage to local filesystem on Windows.
         $ mc cp s3:documents/2014/... C:\backup\2014
```
