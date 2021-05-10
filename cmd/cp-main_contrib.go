// MinIO Object Storage (c) 2021 MinIO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/minio/mc/pkg/probe"
)

// validate the passed metadataString and populate the map
func getMetaDataEntry(metadataString string) (map[string]string, *probe.Error) {
	metaDataMap := make(map[string]string)
	r := strings.NewReader(metadataString)

	type pToken int
	const (
		KEY pToken = iota
		VALUE
	)

	type pState int
	const (
		NORMAL pState = iota
		QSTRING
		DQSTRING
	)

	var key, value strings.Builder

	writeRune := func(ch rune, pt pToken) {
		if pt == KEY {
			key.WriteRune(ch)
		} else if pt == VALUE {
			value.WriteRune(ch)
		} else {
			panic("Invalid parser token type")
		}
	}

	ps := NORMAL
	pt := KEY
	p := 0

	for ; ; p++ {
		ch, _, err := r.ReadRune()
		if err != nil {
			//eof
			if ps == QSTRING || ps == DQSTRING || pt == KEY {
				return nil, probe.NewError(ErrInvalidMetadata)
			}
			metaDataMap[http.CanonicalHeaderKey(key.String())] = value.String()
			return metaDataMap, nil
		}

		if ch == '"' {
			if ps == DQSTRING {
				ps = NORMAL
			} else if ps == QSTRING {
				writeRune(ch, pt)
			} else if ps == NORMAL {
				ps = DQSTRING
			} else {
				break
			}
			continue
		}

		if ch == '\'' {
			if ps == QSTRING {
				ps = NORMAL
			} else if ps == DQSTRING {
				writeRune(ch, pt)
			} else if ps == NORMAL {
				ps = QSTRING
			} else {
				break
			}
			continue
		}

		if ch == '=' {
			if ps == QSTRING || ps == DQSTRING {
				writeRune(ch, pt)
			} else if pt == KEY {
				pt = VALUE
			} else if pt == VALUE {
				writeRune(ch, pt)
			} else {
				break
			}
			continue
		}

		if ch == ';' {
			if ps == QSTRING || ps == DQSTRING {
				writeRune(ch, pt)
			} else if pt == KEY {
				return nil, probe.NewError(ErrInvalidMetadata)
			} else if pt == VALUE {
				metaDataMap[http.CanonicalHeaderKey(key.String())] = value.String()
				key.Reset()
				value.Reset()
				pt = KEY
			} else {
				break
			}
			continue
		}

		writeRune(ch, pt)
	}

	fatalErr := fmt.Sprintf("Invalid parser state at index: %d", p)
	panic(fatalErr)
}
