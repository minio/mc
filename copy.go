package main

import (
	"io"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

func doCopy(methods clientMethods, reader io.ReadCloser, md5hex string, length int64, targetURL string, targetConfig *hostConfig) error {
	writeCloser, err := methods.getTargetWriter(targetURL, targetConfig, md5hex, length)
	if err != nil {
		return iodine.New(err, nil)
	}
	var writers []io.Writer
	writers = append(writers, writeCloser)
	// set up progress bar
	var bar *pb.ProgressBar
	if !globalQuietFlag {
		bar = startBar(length)
		bar.Start()
		writers = append(writers, bar)
	}
	// write progress bar
	multiWriter := io.MultiWriter(writers...)
	// copy data to writers
	_, copyErr := io.CopyN(multiWriter, reader, length)
	// close to see the error, verify it later
	err = writeCloser.Close()
	if copyErr != nil {
		return iodine.New(copyErr, nil)
	}
	if err != nil {
		return iodine.New(err, nil)
	}
	if !globalQuietFlag {
		bar.Finish()
		console.Infoln()
	}
	return nil
}

func doCopySingleSource(methods clientMethods, sourceURL, targetURL string, sourceConfig, targetConfig *hostConfig) error {
	reader, length, md5hex, err := methods.getSourceReader(sourceURL, sourceConfig)
	if err != nil {
		return iodine.New(err, nil)
	}
	return doCopy(methods, reader, md5hex, length, targetURL, targetConfig)
}

func doCopySingleSourceRecursive(methods clientMethods, sourceURL, targetURL string, sourceConfig, targetConfig *hostConfig) error {
	sourceClnt, err := methods.getNewClient(sourceURL, sourceConfig, globalDebugFlag)
	if err != nil {
		return iodine.New(err, nil)
	}
	for itemCh := range sourceClnt.List() {
		if itemCh.Err != nil {
			continue
		}
		newSourceURL, newTargetURL := getNewURLRecursive(sourceURL, targetURL, itemCh.Item.Name)
		doCopySingleSource(methods, newSourceURL, newTargetURL, sourceConfig, targetConfig)
	}
	return nil
}

// doCopyCmd copies objects into and from a bucket or between buckets
func doCopyMultipleSources(methods clientMethods, sourceURLConfigMap map[string]*hostConfig, targetURL string, targetConfig *hostConfig) error {
	sourceURLReaderMap, err := getSourceReaders(methods, sourceURLConfigMap)
	if err != nil {
		return iodine.New(err, nil)
	}
	for sourceURL, sourceReader := range sourceURLReaderMap {
		newTargetURL, err := getNewTargetURL(targetURL, sourceURL)
		if err != nil {
			return iodine.New(err, nil)
		}
		err = doCopy(methods, sourceReader.reader, sourceReader.md5hex, sourceReader.length, newTargetURL, targetConfig)
		if err != nil {
			return iodine.New(err, map[string]string{"Source": sourceURL})
		}
	}
	return nil
}
