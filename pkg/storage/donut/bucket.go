package donut

import (
	"errors"
)

type bucket struct {
	name    string
	objects map[string]Object
}

func (b bucket) ListObjects() (map[string]Object, error) {
	return b.objects, errors.New("Not Implemented")
}

func (b bucket) GetObject(object string) (Object, error) {
	return nil, errors.New("Not Implemented")
}

func (b bucket) GetObjectMetadata(object string) (map[string]string, error) {
	return nil, errors.New("Not Implemented")
}
