package donutclient

import (
	"errors"
	"io"
	"time"

	"github.com/minio-io/mc/pkg/client"
)

type donutClient struct {
	root string
}

// New - instantiates donut client struct
func New(root string) client.Client {
	return &donutClient{root: root}
}

func (d *donutClient) Get(bucket, object string) (body io.ReadCloser, size int64, err error) {
	return nil, 0, errors.New("Not implemented")
}

func (d *donutClient) GetPartial(bucket, key string, offset, length int64) (body io.ReadCloser, size int64, err error) {
	return nil, 0, errors.New("Not implemented")
}

func (d *donutClient) Put(bucket, object string, size int64, body io.Reader) error {
	return errors.New("Not implemented")
}

func (d *donutClient) Stat(bucket, object string) (size int64, date time.Time, err error) {
	return 0, time.Time{}, errors.New("Not implemented")
}

func (d *donutClient) PutBucket(bucket string) error {
	return errors.New("Not implemented")
}

func (d *donutClient) ListBuckets() ([]*client.Bucket, error) {
	return nil, errors.New("Not implemented")
}

func (d *donutClient) ListObjects(bucket string, startAt, prefix, delimiter string, maxKeys int) (items []*client.Item, prefixes []*client.Prefix, err error) {
	return nil, nil, errors.New("Not implemented")
}
