package donut

import "io"

type Donut interface {
	Get(bucket string, object string) (body io.Reader, err error)
	Put(bucket string, object string, size int, body io.Reader) error
	ListObjects(bucket string) (objects map[string]string, err error)
	ListBuckets() (buckets map[string]string, err error)
}
