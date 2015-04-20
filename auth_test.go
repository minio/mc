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

package main

import . "github.com/minio-io/check"

type AuthSuite struct{}

var _ = Suite(&AuthSuite{})

func (s *AuthSuite) TestAuthAccessKeyLength(c *C) {
	// short
	result := isValidSecretKey("123456789012345678901234567890123456789")
	c.Assert(result, Equals, false)

	// long
	result = isValidSecretKey("12345678901234567890123456789012345678901")
	c.Assert(result, Equals, false)

	// 40 characters long
	result = isValidSecretKey("1234567890123456789012345678901234567890")
	c.Assert(result, Equals, true)
}

func (s *AuthSuite) TestValidAccessKeyLength(c *C) {
	// short
	result := isValidAccessKey("1234567890123456789")
	c.Assert(result, Equals, false)

	// long
	result = isValidAccessKey("123456789012345678901")
	c.Assert(result, Equals, false)

	// 40 characters long
	result = isValidAccessKey("12345678901234567890")
	c.Assert(result, Equals, true)

	// alphanumberic characters long
	result = isValidAccessKey("ABCDEFGHIJ1234567890")
	c.Assert(result, Equals, true)

	// alphanumberic characters long
	result = isValidAccessKey("A1B2C3D4E5F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// alphanumberic lower case characters long
	result = isValidAccessKey("a1b2c3d4e5f6g7h8i9j0")
	c.Assert(result, Equals, false)

	// alphanumberic with -
	result = isValidAccessKey("A1B2C3D4-5F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// alphanumberic with .
	result = isValidAccessKey("A1B2C3D4E5F6G7H8I.J0")
	c.Assert(result, Equals, true)

	// alphanumberic with _
	result = isValidAccessKey("A1B2C3D4E_F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// alphanumberic with ~
	result = isValidAccessKey("A1B2C3D4E~F6G7H8I9J0")
	c.Assert(result, Equals, true)

	// with all classes
	result = isValidAccessKey("A1B2C3D4E~F.G_H8~9J0")
	c.Assert(result, Equals, true)

	// with all classes and an extra invalid
	result = isValidAccessKey("A1B2$3D4E~F.G_H8~9J0")
	c.Assert(result, Equals, false)
}
