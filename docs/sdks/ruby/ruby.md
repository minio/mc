# Using AWS SDK for Ruby - Version 2

## Install

### In Ruby 1.9

```
gem install aws-sdk
```

### Example

Next, set up credentials (in e.g. ``~/.aws/credentials``)::

    [default]
    aws_access_key_id = YOUR_MINIO_ACCESS_ID
    aws_secret_access_key = YOUR_MINIO_SECRET_KEY

```
#!/usr/bin/env ruby
require 'aws-sdk'

s3 = Aws::S3::Client.new(endpoint: "http://127.0.0.1:9000/",
                         require_https_for_sse_cpk: false,
                         region: "minio")
resp = s3.list_buckets
resp.buckets.each do |bucket|
  puts "#{bucket.name} => #{bucket.creation_date}"
end
```

NOTE:

    ruby ``aws-sdk`` requires region name should be set, please use any name which
    makes sense. Specifically for this example we choose ``minio``

Grab it here (example-list-buckets.rb)[https://github.com/Minio-io/mc/blob/master/docs/sdks/ruby/example-list-buckets.rb]