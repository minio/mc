package donut

import (
	"errors"
	"path"

	"encoding/json"
	"io/ioutil"
)

type object struct {
	name                string
	objectPath          string
	objectMetadata      map[string]string
	donutObjectMetadata map[string]string
}

// NewObject - instantiate a new object
func NewObject(objectName, p string) (Object, error) {
	if objectName == "" {
		return nil, errors.New("invalid argument")
	}
	o := object{}
	o.name = objectName
	o.objectPath = path.Join(p, objectName)
	return o, nil
}

func (o object) GetObjectName() string {
	return o.name
}

func (o object) GetObjectMetadata() (map[string]string, error) {
	objectMetadata := make(map[string]string)
	objectMetadataBytes, err := ioutil.ReadFile(path.Join(o.objectPath, "objectMetadata.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(objectMetadataBytes, &objectMetadata); err != nil {
		return nil, err
	}
	o.objectMetadata = objectMetadata
	return objectMetadata, nil
}

func (o object) GetDonutObjectMetadata() (map[string]string, error) {
	donutObjectMetadata := make(map[string]string)
	donutObjectMetadataBytes, err := ioutil.ReadFile(path.Join(o.objectPath, "donutObjectMetadata.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(donutObjectMetadataBytes, &donutObjectMetadata); err != nil {
		return nil, err
	}
	o.donutObjectMetadata = donutObjectMetadata
	metadata := make(map[string]string)
	metadata["sys.created"] = o.donutObjectMetadata["created"]
	metadata["sys.md5"] = o.donutObjectMetadata["md5"]
	metadata["sys.size"] = o.donutObjectMetadata["size"]
	return metadata, nil
}
