/*
 * MinIO Client (C) 2020 MinIO, Inc.
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

//go:generate msgp

package cmd

import "time"

type SnapshotEntry struct {
	Key            string    `msg:"k"`
	VersionID      string    `msg:"vid"`
	Size           int64     `msg:"s"`
	ModTime        time.Time `msg:"mt"`
	StorageClass   string    `msg:"sc"`
	ETag           string    `msg:"etag"`
	IsDeleteMarker bool      `msg:"idm"`
	IsLatest       bool      `msg:"il"`
}

type S3Target struct {
	URL          string `json:"url"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	SessionToken string `json:"sessionToken,omitempty"`
	API          string `json:"api"`
	Lookup       string `json:"lookup"`
}
