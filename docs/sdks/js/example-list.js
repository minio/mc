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
