#### access

```go
Name:
   mc access - Set access permissions

USAGE:
   mc access PERMISSION TARGET [TARGET...]

EXAMPLES:

   1. Set bucket to "private" on Amazon S3 cloud storage.
      $ mc access private https://s3.amazonaws.com/burningman2011

   2. Set bucket to "public" on Amazon S3 cloud storage.
      $ mc access public https://s3.amazonaws.com/shared

   3. Set bucket to "authenticated" on Amazon S3 cloud storage to provide read access to IAM Authenticated Users group.
      $ mc access authenticated https://s3.amazonaws.com/shared-authenticated

   4. Set folder to world readwrite (chmod 777) on local filesystem.
      $ mc access public /shared/Music

```
