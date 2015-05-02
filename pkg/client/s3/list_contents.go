/*
 * Minio Client (C) 2015 Minio, Inc.
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

package s3

import "github.com/minio-io/mc/pkg/client"

/// Bucket API operations

// List - list at delimited path not recursive
func (c *s3Client) List() <-chan client.ContentOnChannel {
	contentCh := make(chan client.ContentOnChannel)
	go c.listInGoRoutine(contentCh)
	return contentCh
}

// ListRecursive - list buckets and objects recursively
func (c *s3Client) ListRecursive() <-chan client.ContentOnChannel {
	contentCh := make(chan client.ContentOnChannel)
	go c.listRecursiveInGoRoutine(contentCh)
	return contentCh
}
