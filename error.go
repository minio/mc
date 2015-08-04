package main

import (
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

func ifFatal(err *probe.Error) {
	switch err {
	case nil:
		// nothing to print
	default:
		if !globalDebugFlag {
			console.Fatalln(err.ToError())
		}
		console.Fatalln(err)
	}
}

func ifError(err *probe.Error) {
	switch err {
	case nil:
		// nothing to print
	default:
		if !globalDebugFlag {
			console.Errorln(err.ToError())
			return
		}
		console.Errorln(err)
	}
}
