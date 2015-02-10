# Using AWS SDK for Python

## Install

### In Python 2.6, 2.7, 3.3, 3.4

**WARNING**: Boto 3 is in *developer preview* and **should not** be used in production yet!

```
pip install boto3
```

### Example

Next, set up credentials (in e.g. ``~/.aws/credentials``)::

    [default]
    aws_access_key_id = YOUR_MINIO_ACCESS_ID
    aws_secret_access_key = YOUR_MINIO_SECRET_KEY

```
#!/usr/bin/env python

import boto3
s3 = boto3.resource('s3', use_ssl=False, endpoint_url="http://localhost:9000")
for bucket in s3.buckets.all():
    print(bucket.name)
...
bucket1
bucket2
bucket3
```

Grab it here (example-list-buckets.py)[https://github.com/Minio-io/mc/blob/master/docs/sdks/python/example-list-buckets.py]