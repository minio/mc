package fs

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// Put - upload new object to bucket
func (f *fsClient) Put(bucket, object, md5HexString string, size int64) (io.WriteCloser, error) {
	// Never create directories in here using bucket and object, since such a path might
	// result in files being created as directories, it should be 'PutBucket()'
	// responsibility.

	// bucket and object is deliberately ignored here, since we already have
	// this path, bucket is provided just for compatibility sake at this point
	r, w := io.Pipe()
	blockingWriter := NewBlockingWriteCloser(w)
	go func() {
		// handle md5HexString match internally
		if size < 0 {
			err := iodine.New(client.InvalidArgument{Err: errors.New("invalid argument")}, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		fs, err := os.Create(f.Path)
		if err != nil {
			err := iodine.New(err, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		_, err = io.CopyN(fs, r, size)
		if err != nil {
			err := iodine.New(err, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		blockingWriter.Release(nil)
		r.Close()
	}()
	return blockingWriter, nil
}

// BlockingWriteCloser is a WriteCloser that blocks until released
type BlockingWriteCloser struct {
	w       io.WriteCloser
	release *sync.WaitGroup
	err     error
}

// Write to the underlying writer
func (b *BlockingWriteCloser) Write(p []byte) (int, error) {
	n, err := b.w.Write(p)
	err = iodine.New(err, nil)
	return n, err
}

// Close blocks until another goroutine calls Release(error). Returns error code if either
// writer fails or Release is called with an error.
func (b *BlockingWriteCloser) Close() error {
	err := b.w.Close()
	if err != nil {
		b.err = err
	}
	b.release.Wait()
	return b.err
}

// Release the Close, causing it to unblock. Only call this once. Calling it multiple times results in a panic.
func (b *BlockingWriteCloser) Release(err error) {
	b.release.Done()
	if err != nil {
		b.err = err
	}
}

// NewBlockingWriteCloser Creates a new write closer that must be released by the read consumer.
func NewBlockingWriteCloser(w io.WriteCloser) *BlockingWriteCloser {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &BlockingWriteCloser{w: w, release: wg}
}
