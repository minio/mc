#### access

```go
Name:
   mc access - Set permissions [public, private, readonly] for buckets and folders.

USAGE:
   mc access PERMISSION TARGET [TARGET...]

EXAMPLES:

   1. Set bucket to "private" on Amazon S3 object storage
         $ mc access private https://s3.amazonaws.com/burningman2011

   2. Set bucket to "public" on Amazon S3 object storage
         $ mc access public https://s3.amazonaws.com/shared

   3. Set folder to world readwrite (chmod 777) on local filesystem
         $ mc access public /shared/Music
```
