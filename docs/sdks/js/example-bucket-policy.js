'use strict'

var AWS = require('aws-sdk');

var config = {
  accessKeyId: "MINIO_ACCESS_ID",
  secretAccessKey: "MINIO_SECRET_ID",
  endpoint: "localhost:9000",
  region: "",
  sslEnabled: false,
  s3ForcePathStyle: true
};

AWS.config.update(config);

var s3 = new AWS.S3();

var statement = {
  Sid: "ExampleStatemenent1",
  Effect: "Allow",
  Principal: {
    AWS: "minio::Account-Id:user/Dave"
  },
  Action: [
    "s3.ListBucket",
    "s3.GetObject",
    "s3.PutObject",
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
  Bucket: 'new-bucket',
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
  Bucket: 'new-bucket'
};

s3.getBucketPolicy(params, function(err, data) {
  if (err) {
    console.log(err);
  } else {
    console.log(data);
  }
});
