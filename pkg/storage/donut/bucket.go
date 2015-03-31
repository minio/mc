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
	nodeSlice := 0
	for _, node := range b.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return nil, err
		}
		for _, disk := range disks {
			bucketSlice := fmt.Sprintf("%s$%d$%d", b.name, nodeSlice, disk.GetOrder())
			bucketPath := path.Join(b.donutName, bucketSlice)
			objects, err := disk.ListDir(bucketPath)
			if err != nil {
				return nil, err
			}
			for _, object := range objects {
				b.objects[object.Name()], err = NewObject(object.Name(), path.Join(disk.GetName(), bucketPath))
				if err != nil {
					return nil, err
				}
			}
		}
		nodeSlice = nodeSlice + 1
	}
	return b.objects, nil
}

func (b bucket) decodeData(totalLeft, blockSize int64, readers []io.ReadCloser, encoder Encoder, writer *io.PipeWriter) ([]byte, error) {
	var curBlockSize int64
	if blockSize < totalLeft {
		curBlockSize = blockSize
	} else {
		curBlockSize = totalLeft // cast is safe, blockSize in if protects
	}
	curChunkSize, err := encoder.GetEncodedBlockLen(int(curBlockSize))
	if err != nil {
		return nil, err
	}
	encodedBytes := make([][]byte, 16)
	for i, reader := range readers {
		var bytesBuffer bytes.Buffer
		_, err := io.CopyN(&bytesBuffer, reader, int64(curChunkSize))
		if err != nil {
			return nil, err
		}
		encodedBytes[i] = bytesBuffer.Bytes()
	}
	decodedData, err := encoder.Decode(encodedBytes, int(curBlockSize))
	if err != nil {
		return nil, err
	}
	return decodedData, nil
}

func (b bucket) map2Ints(donutObjectMetadata map[string]string) (totalChunks int, totalLeft, blockSize int64, k, m uint64, err error) {
	totalChunks, err = strconv.Atoi(donutObjectMetadata["chunkCount"])
	totalLeft, err = strconv.ParseInt(donutObjectMetadata["size"], 10, 64)
	blockSize, err = strconv.ParseInt(donutObjectMetadata["blockSize"], 10, 64)
	k, err = strconv.ParseUint(donutObjectMetadata["erasureK"], 10, 8)
	m, err = strconv.ParseUint(donutObjectMetadata["erasureM"], 10, 8)
	return
}

func (b bucket) GetObject(objectName string, writer *io.PipeWriter, donutObjectMetadata map[string]string) {
	if objectName == "" || writer == nil || len(donutObjectMetadata) == 0 {
		writer.CloseWithError(errors.New("invalid argument"))
		return
	}
	expectedMd5sum, err := hex.DecodeString(donutObjectMetadata["md5"])
	if err != nil {
		writer.CloseWithError(err)
		return
	}
	totalChunks, totalLeft, blockSize, k, m, err := b.map2Ints(donutObjectMetadata)
	if err != nil {
		writer.CloseWithError(err)
		return
	}
	technique, ok := donutObjectMetadata["erasureTechnique"]
	if !ok {
		writer.CloseWithError(errors.New("missing erasure Technique"))
		return
	}
	hasher := md5.New()
	mwriter := io.MultiWriter(writer, hasher)
	encoder, err := NewEncoder(uint8(k), uint8(m), technique)
	if err != nil {
		writer.CloseWithError(err)
		return
	}
	readers, err := b.getDiskReaders(objectName, "data")
	if err != nil {
		writer.CloseWithError(err)
		return
	}
	for i := 0; i < totalChunks; i++ {
		decodedData, err := b.decodeData(totalLeft, blockSize, readers, encoder, writer)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		_, err = io.Copy(mwriter, bytes.NewBuffer(decodedData))
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		totalLeft = totalLeft - int64(blockSize)
	}
	actualMd5sum := hasher.Sum(nil)
	if bytes.Compare(expectedMd5sum, actualMd5sum) != 0 {
		writer.CloseWithError(errors.New("checksum mismatch"))
		return
	}
	writer.Close()
	return
}

func (b bucket) getDiskReaders(objectName, objectMeta string) ([]io.ReadCloser, error) {
	readers := make([]io.ReadCloser, 16)
	nodeSlice := 0
	for _, node := range b.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return nil, err
		}
		for _, disk := range disks {
			bucketSlice := fmt.Sprintf("%s$%d$%d", b.name, nodeSlice, disk.GetOrder())
			objectPath := path.Join(b.donutName, bucketSlice, objectName, objectMeta)
			objectSlice, err := disk.OpenFile(objectPath)
			if err != nil {
				return nil, err
			}
			readers[disk.GetOrder()] = objectSlice
		}
		nodeSlice = nodeSlice + 1
	}
	return readers, nil
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
	if objectName == "" {
		return errors.New("invalid argument")
	}
	if contents == nil {
		return errors.New("invalid argument")
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
