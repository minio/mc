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

func ifError(err error) {
	switch e := err.(type) {
	case *probe.Error:
		switch e {
		case nil:
			// nothing to print
		default:
			if !globalDebugFlag {
				console.Errorln(e.ToError())
			}
			console.Errorln(err)
		}
	case nil:
		// nothing to print
	default:
		console.Errorln(err)
	}
}
