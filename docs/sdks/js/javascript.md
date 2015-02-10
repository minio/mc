# Using AWS SDK for Javascript

## Install

### In Node.js

```
npm install aws-sdk
```

### Example

```
var AWS = require('aws-sdk');

var config = {
  accessKeyId: "YOUR_MINIO_ACCESS_ID",
  secretAccessKey: "YOUR_MINIO_SECRET_KEY",
  endpoint: "localhost:9000",
  region: "",
  sslEnabled: false
};

AWS.config.update(config);

var s3 = new AWS.S3();
s3.listBuckets(function(err, data) {
  if (err) {
    console.log(err); // an error occurred
  } else {
    console.log(data); // successful response
  }
});
```