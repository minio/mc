package donut

import (
	"errors"
	"io"
)

type object struct {
	name    string
	readers io.ReadCloser
	writers io.WriteCloser
}

func (o object) GetReader() (io.ReadCloser, error) {
	return nil, errors.New("Not Implemented")
}

func (o object) GetWriter() (io.WriteCloser, error) {
	return nil, errors.New("Not Implemented")
}

func (o object) SetMetadata(metadata map[string]string) error {
	return errors.New("Not Implemented")
}
