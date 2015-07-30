#### mirror

```go
NAME:
   mc mirror - Copy files and folders from a single source to many destinations

USAGE:
   mc mirror SOURCE TARGET [TARGET...]

EXAMPLES:
   1. Cast an object from local filesystem to Amazon S3 cloud storage.
         $ mc mirror star-trek-episode-10-season4.ogg https://s3.amazonaws.com/trekarchive

   2. Cast a bucket recursively from Minio cloud storage to multiple buckets on Amazon S3 cloud storage.
         $ mc mirror https://play.minio.io:9000/photos/2014... https://s3.amazonaws.com/backup-photos https://s3.amazonaws.com/my-photos

   3. Cast a local folder recursively to Minio cloud storage and Amazon S3 cloud storage.
         $ mc mirror backup/... https://play.minio.io:9000/archive https://s3.amazonaws.com/archive

   4. Cast a bucket from aliased Amazon S3 cloud storage to multiple folders on Windows.
         $ mc mirror s3:documents/2014/... C:\backup\2014 C:\shared\volume\backup\2014

   5. Cast a local file of non english character to Amazon s3 cloud storage.
         $ mc mirror 本語/... s3:mylocaldocuments C:\backup\2014 play:backup

```
