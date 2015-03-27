package donut

import (
	"errors"
	"os"
)

type disk struct {
	path   string
	fsType string
}

// NewDisk - instantiate new disk
func NewDisk(path string) (Disk, error) {
	return nil, errors.New("Not Implemented")
}

func (d disk) MakeDir(dirname string) error {
	return errors.New("Not Implemented")
}

func (d disk) ListDir() error {
	return errors.New("Not Implemented")
}

func (d disk) ListFiles(dirname string) error {
	return errors.New("Not Implemented")
}

func (d disk) MakeFile(path string) (*os.File, error) {
	return nil, errors.New("Not Implemented")
}

func (d disk) OpenFile(path string) (*os.File, error) {
	return nil, errors.New("Not Implemented")
}

func (d disk) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (d disk) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
