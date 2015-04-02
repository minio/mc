package main

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client/donut"
)

func upload(urlArg1, urlArg2 *url.URL, mcDonutConfigData *mcDonutConfig) {
	st, stErr := os.Stat(urlArg1.Path)
	if os.IsNotExist(stErr) {
		panic(stErr)
	}
	if st.IsDir() {
		panic("is a directory")
	}
	reader, err := os.OpenFile(urlArg1.Path, 2, os.ModeAppend)
	defer reader.Close()
	if err != nil {
		panic(err)
	}
	nodes := make(map[string][]string)
	for k, v := range mcDonutConfigData.Donuts[urlArg2.Host].Node {
		nodes[k] = v.ActiveDisks
	}
	d, err := donut.GetNewClient(urlArg2.Host, nodes)
	if err != nil {
		fatal(err.Error())
	}
	bucketName, objectName, err := url2Object(urlArg2.String())
	if err != nil {
		fatal(err.Error())
	}
	if err := d.Put(bucketName, objectName, st.Size(), reader); err != nil {
		fatal(err.Error())
	}
}

func download(urlArg1, urlArg2 *url.URL, mcDonutConfigData *mcDonutConfig) {
	nodes := make(map[string][]string)
	for k, v := range mcDonutConfigData.Donuts[urlArg1.Host].Node {
		nodes[k] = v.ActiveDisks
	}
	d, err := donut.GetNewClient(urlArg1.Host, nodes)
	if err != nil {
		fatal(err.Error())
	}

	bucketName, objectName, err := url2Object(urlArg1.String())
	if err != nil {
		fatal(err.Error())
	}
	// Send HEAD request to validate if file exists.
	objectSize, _, err := d.Stat(bucketName, objectName)
	if err != nil {
		fatal(err.Error())
	}
	// Check if the object already exists
	st, err := os.Stat(urlArg2.Path)
	switch os.IsNotExist(err) {
	case true:
		writer, err := os.Create(urlArg2.Path)
		defer writer.Close()
		if err != nil {
			fatal(err.Error())
		}
		reader, size, err := d.Get(bucketName, objectName)
		if err != nil {
			fatal(err.Error())
		}
		_, err = io.CopyN(writer, reader, size)
		if err != nil {
			fatal(err.Error())
		}
		reader.Close()
	case false:
		downloadedSize := st.Size()
		// Verify if file is already downloaded
		if downloadedSize == objectSize {
			msg := fmt.Sprintf("%s object has been already downloaded", urlArg2.Path)
			fatal(msg)
		}
		writer, err := os.OpenFile(urlArg2.Path, os.O_RDWR, 0600)
		defer writer.Close()
		if err != nil {
			fatal(err.Error())
		}
		_, err = writer.Seek(downloadedSize, os.SEEK_SET)
		if err != nil {
			fatal(err.Error())
		}
		remainingSize := objectSize - downloadedSize
		reader, size, err := d.GetPartial(bucketName, objectName, downloadedSize, remainingSize)
		if err != nil {
			fatal(err.Error())
		}
		_, err = io.CopyN(writer, reader, size)
		if err != nil {
			fatal(err.Error())
		}
		reader.Close()
	}
}

func doDonutCPCmd(c *cli.Context) {
	if !c.Args().Present() {
		fatal("no args?")
	}
	mcDonutConfigData, err := loadDonutConfig()
	if err != nil {
		fatal(err.Error())
	}
	switch len(c.Args()) {
	case 2:
		urlArg1, errArg1 := url.Parse(c.Args().Get(0))
		if errArg1 != nil {
			fatal(errArg1.Error())
		}
		urlArg2, errArg2 := url.Parse(c.Args().Get(1))
		if errArg2 != nil {
			fatal(errArg2.Error())
		}
		switch true {
		case urlArg1.Scheme != "" && urlArg2.Scheme == "":
			if _, ok := mcDonutConfigData.Donuts[urlArg1.Host]; !ok {
				msg := fmt.Sprintf("requested donut: <%s> does not exist", urlArg1.Host)
				fatal(msg)
			}
			download(urlArg1, urlArg2, mcDonutConfigData)
		case urlArg1.Scheme == "" && urlArg2.Scheme != "":
			if _, ok := mcDonutConfigData.Donuts[urlArg2.Host]; !ok {
				msg := fmt.Sprintf("requested donut: <%s> does not exist", urlArg2.Host)
				fatal(msg)
			}
			upload(urlArg1, urlArg2, mcDonutConfigData)
		}
	}
}
