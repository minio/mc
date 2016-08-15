/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

package mc

var (
	// MCVersion - version time.RFC3339.
	MCVersion = "DEVELOPMENT.GOGET"
	// MCReleaseTag - release tag in TAG.%Y-%m-%dT%H-%M-%SZ.
	MCReleaseTag = "DEVELOPMENT.GOGET"
	// MCCommitID - latest commit id.
	MCCommitID = "DEVELOPMENT.GOGET"
	// MCShortCommitID - first 12 characters from MCCommitID.
	MCShortCommitID = MCCommitID[:12]
)
