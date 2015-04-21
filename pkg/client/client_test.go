/*
 * Mini Copy, (C) 2015 Minio, Inc.
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

package client

import (
	"testing"

	. "github.com/minio-io/check"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestAuthAccessKeyLength(c *C) {
	// short
	result := IsValidSecretKey("123456789012345678901234567890123456789")
	c.Assert(result, Equals, false)

	// long
	result = IsValidSecretKey("12345678901234567890123456789012345678901")
	c.Assert(result, Equals, false)

	// 40 characters long
	result = IsValidSecretKey("1234567890123456789012345678901234567890")
	c.Assert(result, Equals, true)
}

func (s *MySuite) TestValidAccessKeyLength(c *C) {
	// short
	result := IsValidAccessKey("1234567890123456789")
	c.Assert(result, Equals, false)

	// long
	result = IsValidAccessKey("123456789012345678901")
	c.Assert(result, Equals, false)

	// 40 characters long
	result = IsValidAccessKey("12345678901234567890")
	c.Assert(result, Equals, true)

	// alphanumberic characters long
	result = IsValidAccessKey("ABCDEFGHIJ1234567890")
	c.Assert(result, Equals, true)

	// alphanumberic characters long
	result = IsValidAccessKey("A1B2C3D4E5F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// alphanumberic lower case characters long
	result = IsValidAccessKey("a1b2c3d4e5f6g7h8i9j0")
	c.Assert(result, Equals, false)

	// alphanumberic with -
	result = IsValidAccessKey("A1B2C3D4-5F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// alphanumberic with .
	result = IsValidAccessKey("A1B2C3D4E5F6G7H8I.J0")
	c.Assert(result, Equals, true)

	// alphanumberic with _
	result = IsValidAccessKey("A1B2C3D4E_F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// alphanumberic with ~
	result = IsValidAccessKey("A1B2C3D4E~F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// with all classes
	result = IsValidAccessKey("A1B2C3D4E~F.G_H8~9J0")
	c.Assert(result, Equals, true)

	// with all classes and an extra invalid
	result = IsValidAccessKey("A1B2$3D4E~F.G_H8~9J0")
	c.Assert(result, Equals, false)
}
