#### cat

```go
NAME:
   mc cat - Display contents of a file

USAGE:
   mc cat SOURCE [SOURCE...]

EXAMPLES:
   1. Concantenate an object from Amazon S3 cloud storage to mplayer standard input.
      $ mc cat https://s3.amazonaws.com/ferenginar/klingon_opera_aktuh_maylotah.ogg | mplayer -

   2. Concantenate a file from local filesystem to standard output.
      $ mc cat khitomer-accords.txt

   3. Concantenate multiple files from local filesystem to standard output.
      $ mc cat *.txt > newfile.txt

   4. Concatenate a non english file name from Amazon S3 cloud storage.
      $ mc cat s3:andoria/本語 > /tmp/本語

```
