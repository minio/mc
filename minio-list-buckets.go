package main

import (
	"log"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/minio"
)

func minioDumpBuckets(v []*minio.Bucket) {
	for _, b := range v {
		log.Printf("Bucket :%#v", b)
	}
}

func minioListBuckets(c *cli.Context) {
	hostname, err := getMinioEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	var buckets []*minio.Bucket
	mc := minio.NewMinioClient(hostname)
	buckets, err = mc.Buckets()
	if err != nil {
		log.Fatal(err)
	}

	minioDumpBuckets(buckets)
}
