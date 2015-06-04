/*
 * Minio Client (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this fs except in compliance with the License.
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

package fs

// GenericFileError - generic file error
type GenericFileError struct {
	path string
}

// NotFound (ENOENT) - file not found
type NotFound GenericFileError

func (e NotFound) Error() string {
	return "Requested file ‘" + e.path + "’ not found"
}

// ISFolder (EISDIR) - accessed file is a folder
type ISFolder GenericFileError

func (e ISFolder) Error() string {
	return "Requested file ‘" + e.path + "’ is a folder"
}

// NotFolder (ENOTDIR) - accessed file is not a folder
type NotFolder GenericFileError

func (e NotFolder) Error() string {
	return "Requested file ‘" + e.path + "’ is not a folder"
}
