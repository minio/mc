'use strict'

var AWS = require('aws-sdk');

var config = {
  accessKeyId: "AC5NH40NQLTL4D2W92PM",
  secretAccessKey: "H+AVh8q5G7hEH2r3WxFP135+Q19Aw8yXWel8IGh/HrEjZyTNx/n4Xw==",
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
