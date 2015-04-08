#### copy objects

```
NAME:
   mc cp - copy objects and files

USAGE:
   mc cp [ARGS...] SOURCE TARGET [TARGET...]

DESCRIPTION:
   Copy files and objects recursively between Amazon S3, Minio Object Storage and Filesystem

OPTIONS:
   --recursive, -r	recursively crawls a given directory or bucket

EXAMPLES:
   1. Copy an object from Amazon S3 object storage to local fileystem.
      $ mc cp https://s3.amazonaws.com/jukebox/klingon_opera_aktuh_maylotah.ogg wakeup.ogg

   2. Copy a bucket recursive from Minio object storage to Amazon S3 object storage
      $ mc cp --recursive http://localhost:9000/photos/burningman2011 https://s3.amazonaws.com/burningman/

   3. Copy a local folder to Minio object storage and Amazon S3 object storage
      $ mc cp --recursive backup/ http://localhost:9000/archive/ https://s3.amazonaws.com/archive/

```