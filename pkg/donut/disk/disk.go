package disk

import (
	"io"
	"os"
	"path"
	"strconv"

	"github.com/minio-io/mc/pkg/utils/split"
)

type Disk struct {
	*disk
}

type disk struct {
	root   string
	bucket string
	object string
	chunk  string
	file   *os.File
}

func New(bucket, object string) *Disk {
	d := &Disk{&disk{bucket: bucket, object: object}}
	return d
}

func openFile(path string, flag int) (fl *os.File, err error) {
	fl, err = os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, err
	}
	return fl, nil
}

func (d *Disk) Create() error {
	p := path.Join(d.bucket, d.object, "$"+d.chunk)
	fl, err := openFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL)
	if err != nil {
		return err
	}
	d.file = fl
	return err
}

func (d *Disk) Open() error {
	p := path.Join(d.bucket, d.object, "$"+d.chunk)
	fl, err := openFile(p, os.O_RDONLY)
	if err != nil {
		return err
	}
	d.file = fl
	return err
}

func (d *Disk) Write(b []byte) (n int, err error) {
	if d == nil {
		return 0, os.ErrInvalid
	}
	n, e := d.file.Write(b)
	if n < 0 {
		n = 0
	}
	if n != len(b) {
		err = io.ErrShortWrite
	}
	if e != nil {
		err = e
	}
	return n, err
}

func (d *Disk) Read(b []byte) (n int, err error) {
	if d == nil {
		return 0, os.ErrInvalid
	}
	n, e := d.file.Read(b)
	if n < 0 {
		n = 0
	}
	if n == 0 && len(b) > 0 && e == nil {
		return 0, io.EOF
	}
	if e != nil {
		err = e
	}
	return n, err
}

func (d *Disk) Close() error {
	if d == nil {
		return os.ErrInvalid
	}
	return d.file.Close()
}

func (d *Disk) SplitAndWrite(data io.Reader, chunkSize int) (int, error) {
	if d == nil {
		return 0, os.ErrInvalid
	}
	splits := split.Stream(data, uint64(chunkSize))
	i := 0
	n := 0
	for chunk := range splits {
		if chunk.Err != nil {
			return 0, chunk.Err
		}
		d.chunk = strconv.Itoa(i)
		if err := d.Create(); err != nil {
			return 0, err
		}
		m, err := d.Write(chunk.Data)
		defer d.Close()
		if err != nil {
			return 0, err
		}
		n = n + m
		i = i + 1
	}
	return n, nil
}

func (d *Disk) JoinAndRead() (io.Reader, error) {
	dirname := path.Join(d.root, d.bucket, d.object)
	return split.JoinFiles(dirname, d.object)
}
