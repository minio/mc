package donut

import (
	"errors"
)

type bucket struct {
	name    string
	objects map[string]Object
}

// NewBucket - instantiate a new bucket
func NewBucket(bucketName string) (Bucket, error) {
	if bucketName == "" {
		return nil, errors.New("invalid argument")
	}
	b := bucket{}
	b.name = bucketName
	b.objects = make(map[string]Object)
	return b, nil
}

func (b bucket) GetBucketName() string {
	return b.name
}

func (b bucket) ListObjects() (map[string]Object, error) {
	return b.objects, nil
}

func (b bucket) GetObject(object string) (Object, error) {
	return b.objects[object], nil
}
