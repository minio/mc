package donut

import (
	"errors"
	"fmt"
	"os"
	"path"
	"syscall"

	"io/ioutil"
)

type disk struct {
	path   string
	order  string
	fsType string
}

// NewDisk - instantiate new disk
func NewDisk(diskPath, diskOrder string) (Disk, error) {
	if diskPath == "" || diskOrder == "" {
		return nil, errors.New("invalid argument")
	}
	st, err := os.Stat(diskPath)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, syscall.ENOTDIR
	}
	d := disk{
		path:   diskPath,
		order:  diskOrder,
		fsType: "",
	}
	return d, nil
}

func (d disk) GetDiskName() string {
	return d.path
}

func (d disk) MakeDir(dirname string) error {
	orderedDirname := fmt.Sprintf("%s%s", dirname, d.order)
	return os.MkdirAll(path.Join(d.path, orderedDirname), 0700)
}

func (d disk) ListDir(dirname string) ([]os.FileInfo, error) {
	contents, err := ioutil.ReadDir(path.Join(d.path, dirname))
	if err != nil {
		return nil, err
	}
	var directories []os.FileInfo
	for _, content := range contents {
		// Include only directories, ignore everything else
		if content.IsDir() {
			directories = append(directories, content)
		}
	}
	return directories, nil
}

func (d disk) ListFiles(dirname string) ([]os.FileInfo, error) {
	contents, err := ioutil.ReadDir(path.Join(d.path, dirname))
	if err != nil {
		return nil, err
	}
	var files []os.FileInfo
	for _, content := range contents {
		// Include only regular files, ignore everything else
		if content.Mode().IsRegular() {
			files = append(files, content)
		}
	}
	return files, nil
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
