package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client/donut"
)

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
		mcDonutConfigData, err := loadDonutConfig()
		if err != nil {
			fatal(err.Error())
		}
		if _, ok := mcDonutConfigData.Donuts[urlArg2.Host]; !ok {
			msg := fmt.Sprintf("requested donut: <%s> does not exist", urlArg2.Host)
			fatal(msg)
		}
		nodes := make(map[string][]string)
		for k, v := range mcDonutConfigData.Donuts[urlArg2.Host].Node {
			nodes[k] = v.ActiveDisks
		}
		d, err := donut.GetNewClient(urlArg2.Host, nodes)
		if err != nil {
			fatal(err.Error())
		}
		if err := d.Put(urlArg2.Host, strings.TrimPrefix(urlArg2.Path, "/"), st.Size(), reader); err != nil {
			fatal(err.Error())
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
		mcDonutConfigData, err := loadDonutConfig()
		if err != nil {
			fatal(err.Error())
		}
		if _, ok := mcDonutConfigData.Donuts[urlArg1.Host]; !ok {
			msg := fmt.Sprintf("requested donut: <%s> does not exist", urlArg1.Host)
			fatal(msg)
		}
		nodes := make(map[string][]string)
		for k, v := range mcDonutConfigData.Donuts[urlArg1.Host].Node {
			nodes[k] = v.ActiveDisks
		}
		d, err := donut.GetNewClient(urlArg1.Host, nodes)
		if err != nil {
			fatal(err.Error())
		}
		reader, size, err := d.Get(urlArg1.Host, strings.TrimPrefix(urlArg1.Path, "/"))
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
			download(urlArg1, urlArg2)
		case urlArg1.Scheme == "" && urlArg2.Scheme != "":
			upload(urlArg1, urlArg2)
		}
	}
}
