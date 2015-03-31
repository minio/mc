package donut

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strconv"
)

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

func (b bucket) metadata2Values(donutObjectMetadata map[string]string) (totalChunks int, totalLeft, blockSize int64, k, m uint64, err error) {
	totalChunks, err = strconv.Atoi(donutObjectMetadata["chunkCount"])
	totalLeft, err = strconv.ParseInt(donutObjectMetadata["size"], 10, 64)
	blockSize, err = strconv.ParseInt(donutObjectMetadata["blockSize"], 10, 64)
	k, err = strconv.ParseUint(donutObjectMetadata["erasureK"], 10, 8)
	m, err = strconv.ParseUint(donutObjectMetadata["erasureM"], 10, 8)
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
