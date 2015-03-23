package donut

import "io"

// INTERFACES

// Donut interface
type Donut interface {
	PutBucket(bucket string) error
	Get(bucket, object string) (io.ReadCloser, int64, error)
	Put(bucket, object string) (ObjectWriter, error)
	Stat(bucket, object string) (map[string]string, error)
	ListBuckets() ([]string, error)
	ListObjects(bucket string) ([]string, error)
}

// Bucket interface
type Bucket interface {
	GetNodes() ([]string, error)
}

// Node interface
type Node interface {
	GetBuckets() ([]string, error)
	GetDonutDriverMetadata(bucket, object string) (map[string]string, error)
	GetMetadata(bucket, object string) (map[string]string, error)
	GetReader(bucket, object string) (io.ReadCloser, error)
	GetWriter(bucket, object string) (Writer, error)
	ListObjects(bucket string) ([]string, error)
}

// ObjectWriter interface
type ObjectWriter interface {
	Close() error
	CloseWithError(error) error
	GetMetadata() (map[string]string, error)
	SetMetadata(map[string]string) error
	Write([]byte) (int, error)
}

// Writer interface
type Writer interface {
	ObjectWriter

	GetDonutDriverMetadata() (map[string]string, error)
	SetDonutDriverMetadata(map[string]string) error
}
