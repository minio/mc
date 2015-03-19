'use strict'

var AWS = require('aws-sdk');

var config = {
  accessKeyId: "ECHB22VEKH5I0X4K2T7P",
  secretAccessKey: "IiqhtimkamPZJtV4J8jztm74LpdTSbn7RUASyPzjje2+pfhLJ7nFRg==",
  endpoint: "localhost:9000",
  region: "",
  sslEnabled: false,
  s3ForcePathStyle: true,
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
    "minio:ListBucket",
    "minio:GetObject",
    "minio:PutObject",
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
  Bucket: 'docs',
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
  Bucket: 'docs'
};

s3.getBucketPolicy(params, function(err, data) {
  if (err) {
    console.log(err);
  } else {
    console.log(data);
  }
});
