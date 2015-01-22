package main

import (
	"io"
	"log"
	"os"

	"github.com/minio-io/mc/pkg/minio"
)

func main() {
	newminio := minio.NewMinioClient("127.0.0.1:8080")
	objectReader, objectSize, err := newminio.Get("bucketname", "test")
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.CopyN(os.Stdout, objectReader, objectSize)
	if err != nil {
		log.Fatal(err)
	}

}
