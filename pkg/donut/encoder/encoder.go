package encoder

import (
	"errors"
	"io"

	//	"github.com/minio-io/mc/pkg/encoding/erasure"
	"github.com/minio-io/mc/pkg/donut/disk"
)

type encoder struct{}

const (
	chunkSize = 10 * 1024 * 1024
)

func New() *encoder {
	e := encoder{}
	return &e
}

func (e *encoder) Put(bucket, object string, size int, body io.Reader) error {
	d := disk.New(bucket, object)
	n, err := d.SplitAndWrite(body, chunkSize)
	if err != nil {
		return err
	}
	if n > size {
		return io.ErrUnexpectedEOF
	}
	if n < size {
		return io.ErrShortWrite
	}
	return nil
}

func (e *encoder) Get(bucket, object string) (body io.Reader, err error) {
	d := disk.New(bucket, object)
	return d.JoinAndRead()
}

func (e *encoder) ListObjects(bucket string) (map[string]string, error) {
	return nil, errors.New("Not implemented")
}

func (e *encoder) ListBuckets() (map[string]string, error) {
	return nil, errors.New("Not implemented")
}
