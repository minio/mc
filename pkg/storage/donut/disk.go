package donut

import (
	"errors"
	"os"
	"path"
	"syscall"

	"io/ioutil"
)

type disk struct {
	root   string
	order  int
	fsType string
}

// NewDisk - instantiate new disk
func NewDisk(diskPath string, diskOrder int) (Disk, error) {
	if diskPath == "" || diskOrder < 0 {
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
		root:   diskPath,
		order:  diskOrder,
		fsType: "",
	}
	return d, nil
}

func (d disk) GetName() string {
	return d.root
}

func (d disk) GetOrder() int {
	return d.order
}

func (d disk) GetFileSystemType() string {
	return d.fsType
}

func (d disk) MakeDir(dirname string) error {
	return os.MkdirAll(path.Join(d.root, dirname), 0700)
}

func (d disk) ListDir(dirname string) ([]os.FileInfo, error) {
	contents, err := ioutil.ReadDir(path.Join(d.root, dirname))
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
	contents, err := ioutil.ReadDir(path.Join(d.root, dirname))
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

func (d disk) MakeFile(filename string) (*os.File, error) {
	if filename == "" {
		return nil, errors.New("Invalid argument")
	}
	filePath := path.Join(d.root, filename)
	// Create directories if they don't exist
	if err := os.MkdirAll(path.Dir(filePath), 0700); err != nil {
		return nil, err
	}
	dataFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	return dataFile, nil
}

func (d disk) OpenFile(filename string) (*os.File, error) {
	if filename == "" {
		return nil, errors.New("Invalid argument")
	}
	dataFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return dataFile, nil
}

func (d disk) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (d disk) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
