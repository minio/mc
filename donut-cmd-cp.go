package main

import (
	"io"
	"os"
	"strings"

	"net/url"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client/donut"
)

var d = donut.GetNewClient("testdir")

func upload(urlArg1, urlArg2 *url.URL) {
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
	if urlArg2.Scheme == "donut" {
		err := d.Put(urlArg2.Host, strings.TrimPrefix(urlArg2.Path, "/"), st.Size(), reader)
		if err != nil {
			panic(err)
		}
	}
}

func download(urlArg1, urlArg2 *url.URL) {
	writer, err := os.Create(urlArg2.Path)
	defer writer.Close()
	if err != nil {
		panic(err)
	}
	if urlArg1.Scheme == "donut" {
		reader, size, err := d.Get(urlArg1.Host, strings.TrimPrefix(urlArg1.Path, "/"))
		if err != nil {
			panic(err)
		}
		_, err = io.CopyN(writer, reader, size)
		if err != nil {
			panic(err)
		}
		reader.Close()
	}
}

func doDonutCPCmd(c *cli.Context) {
	os.MkdirAll("testdir", 0755)
	switch len(c.Args()) {
	case 2:
		urlArg1, errArg1 := url.Parse(c.Args().Get(0))
		if errArg1 != nil {
			panic(errArg1)
		}
		urlArg2, errArg2 := url.Parse(c.Args().Get(1))
		if errArg2 != nil {
			panic(errArg2)
		}
		switch true {
		case urlArg1.Scheme != "" && urlArg2.Scheme == "":
			download(urlArg1, urlArg2)
		case urlArg1.Scheme == "" && urlArg2.Scheme != "":
			upload(urlArg1, urlArg2)
		}
	}
}
