// Copyright (c) 2015-2023 MinIO, Inc.
//
// # This file is part of MinIO Object Storage stack
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
	"io"
	"net/http"
	"strconv"
)

type adminPolicyHandler struct {
	endpoint string
	name     string
	policy   []byte
}

func (h adminPolicyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ak := r.Header.Get("Authorization"); len(ak) == 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	switch r.Method {
	case "PUT":
		length, e := strconv.Atoi(r.Header.Get("Content-Length"))
		if e != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var buffer bytes.Buffer
		if _, e = io.CopyN(&buffer, r.Body, int64(length)); e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(h.policy) != buffer.Len() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusForbidden)
	}
}
