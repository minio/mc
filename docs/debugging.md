### Debugging ``mc`` commands

#### mb [DEBUG]
```
$ mc --debug mb localhost:localbucket
PUT /localbucket HTTP/1.1
Host: localhost:9000
User-Agent: Minio s3Client
Content-Length: 0
Authorization: AWS **PASSWORD**STRIPPED**
Date: Wed, 08 Apr 2015 23:44:29 GMT
Accept-Encoding: gzip

HTTP/1.1 200 OK
```

#### ls [DEBUG]
```
$ mc --debug ls localhost:
GET / HTTP/1.1
Host: localhost:9000
User-Agent: Minio s3Client
Authorization: AWS **PASSWORD**STRIPPED**
Date: Wed, 08 Apr 2015 23:42:36 GMT
Accept-Encoding: gzip

HTTP/1.1 200 OK
Connection: close
Content-Length: 221
Accept-Ranges: bytes
Content-Type: application/xml
Date: Wed, 08 Apr 2015 23:42:36 GMT
Server: Minio

2015-04-08 09:42:36 PDT               newbucket
```