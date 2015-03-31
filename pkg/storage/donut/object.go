package donut

import (
	"errors"
)

type object struct {
	name     string
	metadata map[string]string
}

// NewObject - instantiate a new object
func NewObject(objectName string) (Object, error) {
	if objectName == "" {
		return nil, errors.New("invalid argument")
	}
	o := object{}
	o.name = objectName
	o.metadata = make(map[string]string)
	return o, nil
}

func (o object) GetObjectName() string {
	return o.name
}

func (o object) GetMetadata() (map[string]string, error) {
	return nil, errors.New("Not Implemented")
}
