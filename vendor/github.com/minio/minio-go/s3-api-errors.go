/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package minio

import "net/http"

// TODO - handle this automatically by re-writing the request as virtual style.
//
// For path style requests on buckets with wrong endpoint s3 returns back a
// generic error. Following block of code tries to make this meaningful for
// the user by fetching the proper endpoint. Additionally also sets AmzBucketRegion.
func (a s3API) handleStatusMovedPermanently(resp *http.Response, bucket, object string) ErrorResponse {
	errorResponse := ErrorResponse{
		Code:            "PermanentRedirect",
		RequestID:       resp.Header.Get("x-amz-request-id"),
		HostID:          resp.Header.Get("x-amz-id-2"),
		AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
	}
	errorResponse.Resource = separator + bucket
	if object != "" {
		errorResponse.Resource = separator + bucket + separator + object
	}
	var endPoint string
	if errorResponse.AmzBucketRegion != "" {
		region := errorResponse.AmzBucketRegion
		endPoint = getEndpoint(region)
	} else {
		region, err := a.getBucketLocation(bucket)
		if err != nil {
			return *ToErrorResponse(err)
		}
		endPoint = getEndpoint(region)
	}
	msg := "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint."
	errorResponse.Message = msg
	return errorResponse
}
