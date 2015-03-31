package donut

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"time"

	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"github.com/minio-io/minio/pkg/utils/split"
)

type bucket struct {
	name      string
	donutName string
	nodes     map[string]Node
	objects   map[string]Object
}

// NewBucket - instantiate a new bucket
func NewBucket(bucketName, donutName string, nodes map[string]Node) (Bucket, error) {
	if bucketName == "" {
		return nil, errors.New("invalid argument")
	}
	b := bucket{}
	b.name = bucketName
	b.donutName = donutName
	b.objects = make(map[string]Object)
	b.nodes = nodes
	return b, nil
}

func (b bucket) ListNodes() (map[string]Node, error) {
	return b.nodes, nil
}

func (b bucket) GetBucketName() string {
	return b.name
}

func (b bucket) ListObjects() (map[string]Object, error) {
	return b.objects, nil
}

func (b bucket) GetObject(object string) (io.ReadCloser, error) {
	var err error
	b.objects[object], err = NewObject(object)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (b bucket) SetObjectMetadata(objectName string, objectMetadata map[string]string) error {
	if len(objectMetadata) == 0 {
		return errors.New("invalid argument")
	}
	objectMetadataWriters, err := b.getDiskWriters(objectName, "objectMetadata.json")
	if err != nil {
		return err
	}
	for _, objectMetadataWriter := range objectMetadataWriters {
		defer objectMetadataWriter.Close()
	}
	for _, objectMetadataWriter := range objectMetadataWriters {
		jenc := json.NewEncoder(objectMetadataWriter)
		if err := jenc.Encode(objectMetadata); err != nil {
			return err
		}
	}
	return nil
}

func (b bucket) SetDonutObjectMetadata(objectName string, donutObjectMetadata map[string]string) error {
	if len(donutObjectMetadata) == 0 {
		return errors.New("invalid argument")
	}
	donutObjectMetadataWriters, err := b.getDiskWriters(objectName, "donutObjectMetadata.json")
	if err != nil {
		return err
	}
	for _, donutObjectMetadataWriter := range donutObjectMetadataWriters {
		defer donutObjectMetadataWriter.Close()
	}
	for _, donutObjectMetadataWriter := range donutObjectMetadataWriters {
		jenc := json.NewEncoder(donutObjectMetadataWriter)
		if err := jenc.Encode(donutObjectMetadata); err != nil {
			return err
		}
	}
	return nil
}

func (b bucket) getDiskWriters(objectName, objectMeta string) ([]io.WriteCloser, error) {
	writers := make([]io.WriteCloser, 16)
	nodeSlice := 0
	for _, node := range b.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return nil, err
		}
		for _, disk := range disks {
			bucketSlice := fmt.Sprintf("%s$%d$%d", b.name, nodeSlice, disk.GetOrder())
			objectPath := path.Join(b.donutName, bucketSlice, objectName, objectMeta)
			objectSlice, err := disk.MakeFile(objectPath)
			if err != nil {
				return nil, err
			}
			writers[disk.GetOrder()] = objectSlice
		}
		nodeSlice = nodeSlice + 1
	}
	return writers, nil
}

func (b bucket) PutObject(objectName string, contents io.ReadCloser) error {
	var err error
	b.objects[objectName], err = NewObject(objectName)
	if err != nil {
		return err
	}

	writers, err := b.getDiskWriters(objectName, "data")
	if err != nil {
		return err
	}
	for _, writer := range writers {
		defer writer.Close()
	}

	chunks := split.Stream(contents, 10*1024*1024)
	encoder, err := NewEncoder(8, 8, "Cauchy")
	if err != nil {
		return err
	}
	chunkCount := 0
	totalLength := 0
	summer := md5.New()
	for chunk := range chunks {
		if chunk.Err == nil {
			totalLength = totalLength + len(chunk.Data)
			encodedBlocks, _ := encoder.Encode(chunk.Data)
			summer.Write(chunk.Data)
			for blockIndex, block := range encodedBlocks {
				io.Copy(writers[blockIndex], bytes.NewBuffer(block))
			}
		}
		chunkCount = chunkCount + 1
	}

	dataMd5sum := summer.Sum(nil)
	donutObjectMetadata := make(map[string]string)
	donutObjectMetadata["blockSize"] = strconv.Itoa(10 * 1024 * 1024)
	donutObjectMetadata["chunkCount"] = strconv.Itoa(chunkCount)
	donutObjectMetadata["created"] = time.Now().Format(time.RFC3339Nano)
	donutObjectMetadata["erasureK"] = "8"
	donutObjectMetadata["erasureM"] = "8"
	donutObjectMetadata["erasureTechnique"] = "Cauchy"
	donutObjectMetadata["md5"] = hex.EncodeToString(dataMd5sum)
	donutObjectMetadata["size"] = strconv.Itoa(totalLength)
	if err := b.SetDonutObjectMetadata(objectName, donutObjectMetadata); err != nil {
		return err
	}
	objectMetadata := make(map[string]string)
	objectMetadata["bucket"] = b.name
	objectMetadata["object"] = objectName
	objectMetadata["contentType"] = "application/octet-stream"
	if err := b.SetObjectMetadata(objectName, objectMetadata); err != nil {
		return err
	}
	return nil
}
