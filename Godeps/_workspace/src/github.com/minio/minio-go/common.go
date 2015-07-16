/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// decoder provides a unified decoding method interface
type decoder interface {
	Decode(v interface{}) error
}

// acceptTypeDecoder provide decoded value in given acceptType
func acceptTypeDecoder(body io.Reader, acceptType string, v interface{}) error {
	var decoder decoder
	switch {
	case acceptType == "application/xml":
		decoder = xml.NewDecoder(body)
	case acceptType == "application/json":
		decoder = json.NewDecoder(body)
	default:
		decoder = xml.NewDecoder(body)
	}
	return decoder.Decode(v)
}

// urlEncodedName encode the strings from UTF-8 byte representations to HTML hex escape sequences
//
// This is necessary since regular url.Parse() and url.Encode() functions do not support UTF-8
// non english characters cannot be parsed due to the nature in which url.Encode() is written
//
// This function on the other hand is a direct replacement for url.Encode() technique to support
// pretty much every UTF-8 character.
func urlEncodeName(name string) (string, error) {
	// if object matches reserved string, no need to encode them
	reservedNames := regexp.MustCompile("^[a-zA-Z0-9-_.~/]+$")
	if reservedNames.MatchString(name) {
		return name, nil
	}
	var encodedName string
	for _, s := range name {
		if 'A' <= s && s <= 'Z' || 'a' <= s && s <= 'z' || '0' <= s && s <= '9' { // §2.3 Unreserved characters (mark)
			encodedName = encodedName + string(s)
			continue
		}
		switch s {
		case '-', '_', '.', '~', '/': // §2.3 Unreserved characters (mark)
			encodedName = encodedName + string(s)
			continue
		default:
			len := utf8.RuneLen(s)
			if len < 0 {
				return "", errors.New("invalid utf-8")
			}
			u := make([]byte, len)
			utf8.EncodeRune(u, s)
			for _, r := range u {
				hex := hex.EncodeToString([]byte{r})
				encodedName = encodedName + "%" + strings.ToUpper(hex)
			}
		}
	}
	return encodedName, nil
}

// sum256Reader calculate sha256 sum for an input read seeker
func sum256Reader(reader io.ReadSeeker) ([]byte, error) {
	h := sha256.New()
	var err error

	start, _ := reader.Seek(0, 1)
	defer reader.Seek(start, 0)

	for err == nil {
		length := 0
		byteBuffer := make([]byte, 1024*1024)
		length, err = reader.Read(byteBuffer)
		byteBuffer = byteBuffer[0:length]
		h.Write(byteBuffer)
	}

	if err != io.EOF {
		return nil, err
	}

	return h.Sum(nil), nil
}

// sum256 calculate sha256 sum for an input byte array
func sum256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

// sumHMAC calculate hmac between two input byte array
func sumHMAC(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

// sumMD5Reader calculate md5 for an input read seeker of a given size
func sumMD5Reader(body io.ReadSeeker, size int64) ([]byte, error) {
	hasher := md5.New()
	_, err := io.CopyN(hasher, body, size)
	if err != nil {
		return nil, err
	}
	// seek back
	_, err = body.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	// encode the md5 checksum in base64
	return hasher.Sum(nil), nil
}
