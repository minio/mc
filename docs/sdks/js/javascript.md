# Using AWS SDK for Javascript

## Install

### In Node.js

```
npm install aws-sdk
```

### Example ``GetService`` and ``ListObjects``

```
'use strict'

var AWS = require('aws-sdk');

var config = {
  accessKeyId: "MINIO_ACCESS_ID",
  secretAccessKey: "MINIO_SECRET_ID",
  endpoint: "localhost:9000",
  region: "",
  sslEnabled: false,
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


var params = {
  Bucket: "your-bucket"
};

s3.listObjects(params, function(err, data) {
  if (err) console.log(err);
  else console.log(data);
});
```

Grab it here [example-list.js](https://github.com/Minio-io/mc/blob/master/docs/sdks/js/example-list.js)

### Example ``GetBucketPolicy`` and ``PutBucketPolicy``

```
'use strict'

var AWS = require('aws-sdk');

var config = {
  accessKeyId: "MINIO_ACCESS_ID",
  secretAccessKey: "MINIO_SECRET_ID",
  endpoint: "localhost:9000",
  region: "",
  sslEnabled: false,
};

AWS.config.update(config);

var s3 = new AWS.S3();

var statement = {
  Sid: "ExampleStatemenent1",
  Effect: "Allow",
  Principal: {
    AWS: "minio::1111111:murphy"
  },
  Action: [
    "s3:ListBucket",
    "s3:GetObject",
    "s3:PutObject",
  ],
  Resource: [
    "minio:::examplebucket"
  ]
}

var policy = {
  Version: "2012-10-20",
  Statement: [statement],
}

var params = {
  Bucket: 'newbucket',
  Policy: JSON.stringify(policy),
}

s3.putBucketPolicy(params, function(err, data) {
  if (err) {
    console.log(err);
  } else {
    console.log(data);
  }
});

var params = {
  Bucket: 'newbucket'
};

s3.getBucketPolicy(params, function(err, data) {
  if (err) {
    console.log(err);
  } else {
    console.log(data);
  }
});
```

Grab it here [example-bucket-policy.js](https://github.com/Minio-io/mc/blob/master/docs/sdks/js/example-bucket-policy.js)
