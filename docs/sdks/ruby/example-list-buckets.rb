#!/usr/bin/env ruby
require 'aws-sdk'

s3 = Aws::S3::Client.new(endpoint: "http://localhost:9000/",
                         require_https_for_sse_cpk: false,
                         region: "minio")
resp = s3.list_buckets
resp.buckets.each do |bucket|
  puts "#{bucket.name} => #{bucket.creation_date}"
end
