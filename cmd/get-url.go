// Copyright (c) 2015-2024 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
)

// prepareGetURLs - prepares target and source clientURLs for copying.
func prepareGetURLs(ctx context.Context, o prepareCopyURLsOpts) chan URLs {
	copyURLsCh := make(chan URLs)
	go func(o prepareCopyURLsOpts) {
		defer close(copyURLsCh)
		copyURLsContent, err := guessGetURLType(ctx, o)
		if err != nil {
			copyURLsCh <- URLs{Error: err}
			return
		}

		switch copyURLsContent.copyType {
		case copyURLsTypeA:
			copyURLsCh <- prepareCopyURLsTypeA(ctx, *copyURLsContent, o)
		case copyURLsTypeB:
			copyURLsCh <- prepareCopyURLsTypeB(ctx, *copyURLsContent, o)
		default:
			copyURLsCh <- URLs{Error: errInvalidArgument().Trace(o.sourceURLs...)}
		}
	}(o)

	finalCopyURLsCh := make(chan URLs)
	go func() {
		defer close(finalCopyURLsCh)
		for cpURLs := range copyURLsCh {
			if cpURLs.Error != nil {
				finalCopyURLsCh <- cpURLs
				continue
			}
			finalCopyURLsCh <- cpURLs
		}
	}()

	return finalCopyURLsCh
}

// guessGetURLType guesses the type of clientURL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessGetURLType(ctx context.Context, o prepareCopyURLsOpts) (*copyURLsContent, *probe.Error) {
	cc := new(copyURLsContent)

	// Extract alias before fiddling with the clientURL.
	cc.sourceURL = o.sourceURLs[0]
	cc.sourceAlias, _, _ = mustExpandAlias(cc.sourceURL)
	// Find alias and expanded clientURL.
	cc.targetAlias, cc.targetURL, _ = mustExpandAlias(o.targetURL)

	if len(o.sourceURLs) == 1 { // 1 Source, 1 Target
		var err *probe.Error

		client, err := newClient(cc.sourceURL)
		if err != nil {
			cc.copyType = copyURLsTypeInvalid
			return cc, err
		}
		s3clnt, ok := client.(*S3Client)
		if !ok {
			return cc, probe.NewError(fmt.Errorf("Source is not s3."))
		}
		bucket, path := s3clnt.url2BucketAndObject()
		if bucket == "" {
			return cc, probe.NewError(fmt.Errorf("Please set bucket for s3 resource."))
		}
		if path == "" {
			return cc, probe.NewError(fmt.Errorf("Please set a full path for s3 resource."))
		}
		cc.sourceContent = s3clnt.objectInfo2ClientContent(bucket, minio.ObjectInfo{
			Key: path, VersionID: o.versionID,
		})

		client, err = newClient(o.targetURL)
		if err != nil {
			cc.copyType = copyURLsTypeInvalid
			return cc, err
		}
		_, ok = client.(*fsClient)
		if !ok {
			return cc, probe.NewError(fmt.Errorf("Target is not local filesystem."))
		}

		// If target is a folder, it is Type B.
		var isDir bool
		isDir, cc.targetContent = isAliasURLDir(ctx, o.targetURL, o.encKeyDB, o.timeRef, o.ignoreBucketExistsCheck)
		if isDir {
			cc.copyType = copyURLsTypeB
			cc.sourceVersionID = cc.sourceContent.VersionID
			return cc, nil
		}

		// else Type A.
		cc.copyType = copyURLsTypeA
		cc.sourceVersionID = cc.sourceContent.VersionID
		return cc, nil
	}

	cc.copyType = copyURLsTypeInvalid
	return cc, errInvalidArgument().Trace()
}
