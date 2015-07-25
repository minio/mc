#### cp

```go
NAME:
   mc cp - Copy files and folders from many sources to a single destination

USAGE:
   mc cp SOURCE [SOURCE...] TARGET

EXAMPLES:
   1. Copy list of objects from local file system to Amazon S3 cloud storage.
         $ mc cp Music/*.ogg https://s3.amazonaws.com/jukebox/

   2. Copy a bucket recursively from Minio cloud storage to Amazon S3 cloud storage.
         $ mc cp https://play.minio.io:9000/photos/burningman2011... https://s3.amazonaws.com/private-photos/burningman/

   3. Copy multiple local folders recursively to Minio cloud storage.
         $ mc cp backup/2014/... backup/2015/... https://play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 cloud storage to local filesystem on Windows.
         $ mc cp s3:documents/2014/... C:\backup\2014

   5. Copy an object of non english characters to Amazon S3 cloud storage.
         $ mc cp 本語 s3:andoria/本語

```
