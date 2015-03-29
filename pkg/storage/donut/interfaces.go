package donut

import (
	"io"
	"os"
)

// Collection of Donut specification interfaces

// Donut interface
type Donut interface {
	MakeBucket(bucket string) error
	ListBuckets() (map[string]Bucket, error)

	Heal() error
	Rebalance() error
	Info() error

	AttachNode(node Node) error
	DetachNode(node Node) error

	SaveConfig() ([]byte, error)
	LoadConfig([]byte) error
}

// Encoder interface
type Encoder interface {
	Encode(data []byte) (encodedData [][]byte, err error)
	Decode(encodedData [][]byte, dataLength int) (data []byte, err error)
}

// Bucket interface
type Bucket interface {
	GetBucketName() string

	ListObjects() (map[string]Object, error)
	GetObject(object string) (Object, error)
}

// Object interface
type Object interface {
	GetReader() (io.ReadCloser, error)
	GetWriter() (io.WriteCloser, error)

	GetObjectName() string
	SetMetadata(map[string]string) error
	GetMetadata() (map[string]string, error)
}

// Node interface
type Node interface {
	ListDisks() (map[string]Disk, error)
	AttachDisk(disk Disk) error
	DetachDisk(disk Disk) error

	GetNodeName() string
	SaveConfig() ([]byte, error)
	LoadConfig([]byte) error
}

// Disk interface
type Disk interface {
	MakeDir(dirname string) error

	ListDir(dirname string) ([]os.FileInfo, error)
	ListFiles(dirname string) ([]os.FileInfo, error)

	MakeFile(path string) (*os.File, error)
	OpenFile(path string) (*os.File, error)

	GetDiskName() string
	SaveConfig() ([]byte, error)
	LoadConfig([]byte) error
}
