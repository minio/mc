package main

import (
	"io"
	"os"
	"strings"

	"net/url"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/donut"
)

func doDonutCPCmd(c *cli.Context) {
	var e donut.Donut
	e = donut.NewDonut("testdir")
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
			writer, err := os.Create(urlArg2.Path)
			defer writer.Close()
			if err != nil {
				panic(err)
			}
			if urlArg1.Scheme == "donut" {
				e.CreateBucket(urlArg1.Host)
				reader, err := e.GetObjectReader(urlArg1.Host, strings.TrimPrefix(urlArg1.Path, "/"))
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(writer, reader)
				if err != nil {
					panic(err)
				}
				reader.Close()
			}
		case urlArg1.Scheme == "" && urlArg2.Scheme != "":
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
				e.CreateBucket(urlArg2.Host)
				writer, err := e.GetObjectWriter(urlArg2.Host, strings.TrimPrefix(urlArg2.Path, "/"))
				if err != nil {
					panic(err)
				}
				io.Copy(writer, reader)
				writer.Close()
			}
		}
	}
}
