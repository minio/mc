#### cat

```go
NAME:
   mc cat - Concantenate objects or files to standard output

USAGE:
   mc cat SOURCE [SOURCE...]

EXAMPLES:
   1. Concantenate an object from Amazon S3 object storage to mplayer standard input
         $ mc cat https://s3.amazonaws.com/ferenginar/klingon_opera_aktuh_maylotah.ogg | mplayer -

   2. Concantenate a file from local filesystem to standard output.
         $ mc cat khitomer-accords.txt

   3. Concantenate multiple files from local filesystem to standard output.
         $ mc cat *.txt > newfile.txt
```
