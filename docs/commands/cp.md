#### cp

```go
NAME:
   mc cp - Copy objects and files

USAGE:
   mc cp SOURCE TARGET [TARGET...]

EXAMPLES:
   1. Copy an object from Amazon S3 object storage to local fileystem.
      $ mc cp Music/*.ogg https://s3.amazonaws.com/jukebox/

   2. Copy a bucket recursively from Minio object storage to Amazon S3 object storage
      $ mc cp http://localhost:9000/photos/burningman2011... https://s3.amazonaws.com/burningman/

   3. Copy a local folder recursively to Minio object storage and Amazon S3 object storage
      $ mc cp backup/... http://localhost:9000/archive/

   4. Copy an object from Amazon S3 object storage to local filesystem on Windows.
      $ mc cp s3:documents/2014/... backup/2014

```