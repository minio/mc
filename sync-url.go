package main

import (
	"errors"

	"github.com/minio/minio/pkg/iodine"
)

type syncURLs copyURLs

// prepareCopyURLs - prepares target and source URLs for syncing.
func prepareSyncURLs(sourceURL string, targetURLs []string) <-chan *copyURLs {
	syncURLsCh := make(chan *copyURLs)

	go func() {
		defer close(syncURLsCh)
		for _, targetURL := range targetURLs {
			switch guessCopyURLType([]string{sourceURL}, targetURL) {
			case copyURLsTypeA:
				syncURLs := prepareCopyURLsTypeA(sourceURL, targetURL)
				syncURLsCh <- syncURLs
			case copyURLsTypeB:
				syncURLs := prepareCopyURLsTypeB(sourceURL, targetURL)
				syncURLsCh <- syncURLs
			case copyURLsTypeC:
				for syncURLs := range prepareCopyURLsTypeC(sourceURL, targetURL) {
					syncURLsCh <- syncURLs
				}
			default:
				syncURLsCh <- &copyURLs{Error: iodine.New(errors.New("Invalid arguments."), nil)}
			}
		}
	}()
	return syncURLsCh
}
