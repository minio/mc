// Copyright (c) 2015-2022 MinIO, Inc.
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
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"

	checkv1 "gopkg.in/check.v1"
)

// Test list files in a folder.
func (s *TestSuite) TestList(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	// Create multiple files.
	objectPath := filepath.Join(root, "object1")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	data := "hello"

	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	},
	)
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))

	objectPath = filepath.Join(root, "object2")
	fsClient, err = fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))

	fsClient, err = fsNew(root)
	c.Assert(err, checkv1.IsNil)

	// Verify previously create files and list them.
	var contents []*ClientContent
	for content := range fsClient.List(globalContext, ListOptions{ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	c.Assert(err, checkv1.IsNil)
	c.Assert(len(contents), checkv1.Equals, 1)
	c.Assert(contents[0].Type.IsDir(), checkv1.Equals, true)

	// Create another file.
	objectPath = filepath.Join(root, "test1/newObject1")
	fsClient, err = fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))

	fsClient, err = fsNew(root)
	c.Assert(err, checkv1.IsNil)

	contents = nil
	// List non recursive to list only top level files.
	for content := range fsClient.List(globalContext, ListOptions{ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	c.Assert(err, checkv1.IsNil)
	c.Assert(len(contents), checkv1.Equals, 1)
	c.Assert(contents[0].Type.IsDir(), checkv1.Equals, true)

	fsClient, err = fsNew(root)
	c.Assert(err, checkv1.IsNil)

	contents = nil
	// List recursively all files and verify.
	for content := range fsClient.List(globalContext, ListOptions{Recursive: true, ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}

	c.Assert(err, checkv1.IsNil)
	c.Assert(len(contents), checkv1.Equals, 3)

	var regularFiles int
	var regularDirs int
	// Test number of expected files and directories.
	for _, content := range contents {
		if content.Type.IsRegular() {
			regularFiles++
			continue
		}
		if content.Type.IsDir() {
			regularDirs++
			continue
		}
	}
	c.Assert(regularDirs, checkv1.Equals, 0)
	c.Assert(regularFiles, checkv1.Equals, 3)

	// Create an ignored file and list to verify if its ignored.
	objectPath = filepath.Join(root, "test1/.DS_Store")
	fsClient, err = fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))

	fsClient, err = fsNew(root)
	c.Assert(err, checkv1.IsNil)

	contents = nil
	// List recursively all files and verify.
	for content := range fsClient.List(globalContext, ListOptions{Recursive: true, ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}

	c.Assert(err, checkv1.IsNil)
	switch runtime.GOOS {
	case "darwin":
		c.Assert(len(contents), checkv1.Equals, 3)
	default:
		c.Assert(len(contents), checkv1.Equals, 4)
	}

	regularFiles = 0
	// Test number of expected files.
	for _, content := range contents {
		if content.Type.IsRegular() {
			regularFiles++
			continue
		}
	}
	switch runtime.GOOS {
	case "darwin":
		c.Assert(regularFiles, checkv1.Equals, 3)
	default:
		c.Assert(regularFiles, checkv1.Equals, 4)
	}
}

// Test put bucket aka 'mkdir()' operation.
func (s *TestSuite) TestPutBucket(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fsNew(bucketPath)
	c.Assert(err, checkv1.IsNil)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, checkv1.IsNil)
}

// Test stat bucket aka 'stat()' operation.
func (s *TestSuite) TestStatBucket(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsClient, err := fsNew(bucketPath)
	c.Assert(err, checkv1.IsNil)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, checkv1.IsNil)
	_, err = fsClient.Stat(context.Background(), StatOptions{})
	c.Assert(err, checkv1.IsNil)
}

// Test bucket acl fails for directories.
func (s *TestSuite) TestBucketACLFails(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fsNew(bucketPath)
	c.Assert(err, checkv1.IsNil)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, checkv1.IsNil)

	// On windows setting permissions is not supported.
	if runtime.GOOS != "windows" {
		err = fsClient.SetAccess(context.Background(), "readonly", false)
		c.Assert(err, checkv1.IsNil)

		_, _, err = fsClient.GetAccess(context.Background())
		c.Assert(err, checkv1.IsNil)
	}
}

// Test creating a file.
func (s *TestSuite) TestPut(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	data := "hello"
	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	},
	)

	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))
}

// Test read a file.
func (s *TestSuite) TestGet(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	data := "hello"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))

	reader, _, err = fsClient.Get(context.Background(), GetOptions{})
	c.Assert(err, checkv1.IsNil)
	var results bytes.Buffer
	_, e = io.Copy(&results, reader)
	c.Assert(e, checkv1.IsNil)
	c.Assert([]byte(data), checkv1.DeepEquals, results.Bytes())
}

// Test get range in a file.
func (s *TestSuite) TestGetRange(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	data := "hello world"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))

	reader, _, err = fsClient.Get(context.Background(), GetOptions{})
	c.Assert(err, checkv1.IsNil)
	var results bytes.Buffer
	buf := make([]byte, 5)
	m, e := reader.(io.ReaderAt).ReadAt(buf, 0)
	c.Assert(e, checkv1.IsNil)
	c.Assert(m, checkv1.Equals, 5)
	_, e = results.Write(buf)
	c.Assert(e, checkv1.IsNil)
	c.Assert([]byte("hello"), checkv1.DeepEquals, results.Bytes())
}

// Test stat file.
func (s *TestSuite) TestStatObject(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, checkv1.IsNil)

	data := "hello"
	dataLen := len(data)
	reader := bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(dataLen), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	},
	)
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))

	content, err := fsClient.Stat(context.Background(), StatOptions{})
	c.Assert(err, checkv1.IsNil)
	c.Assert(content.Size, checkv1.Equals, int64(dataLen))
}

// Test copy.
func (s *TestSuite) TestCopy(c *checkv1.C) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	c.Assert(e, checkv1.IsNil)
	defer os.RemoveAll(root)
	sourcePath := filepath.Join(root, "source")
	targetPath := filepath.Join(root, "target")
	fsClientTarget, err := fsNew(targetPath)
	c.Assert(err, checkv1.IsNil)
	fsClientSource, err := fsNew(sourcePath)
	c.Assert(err, checkv1.IsNil)

	data := "hello world"
	reader := bytes.NewReader([]byte(data))
	n, err := fsClientSource.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(data)))
	err = fsClientTarget.Copy(context.Background(), sourcePath, CopyOptions{size: int64(len(data))}, nil)
	c.Assert(err, checkv1.IsNil)
}
