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
	"context"
	checkv1 "gopkg.in/check.v1"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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
	switch {
	case r.Method == "PUT":
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

func (s *TestSuite) TestAdminSTSOperation(c *checkv1.C) {
	sts := stsHandler{
		endpoint: "/",
		jwt:      []byte("eyJhbGciOiJSUzI1NiIsImtpZCI6Inc0dFNjMEc5Tk0wQWhGaWJYaWIzbkpRZkRKeDc1dURRTUVpOTNvTHJ0OWcifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzMxMTg3NzEwLCJpYXQiOjE2OTk2NTE3MTAsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJtaW5pby10ZW5hbnQtMSIsInBvZCI6eyJuYW1lIjoic2V0dXAtYnVja2V0LXQ4eGdjIiwidWlkIjoiNjZhYjlkZWItNzkwMC00YTFlLTgzMDgtMTkwODIwZmQ3NDY5In0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJtYy1qb2Itc2EiLCJ1aWQiOiI3OTc4NzJjZC1kMjkwLTRlM2EtYjYyMC00ZGFkYzZhNzUyMTYifSwid2FybmFmdGVyIjoxNjk5NjU1MzE3fSwibmJmIjoxNjk5NjUxNzEwLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6bWluaW8tdGVuYW50LTE6bWMtam9iLXNhIn0.rY7dpAh8GBTViH9Ges7tRhgyihdFWEN0DwXchelmZg58VOI526S-YfbCqrxksTs8Iu0fp1rmk1cUj7FGDh3AOv2RphHjoWci1802zKkHgH0iOEbKMp3jHXwfyHda8CyrSCPycGzClueCf1ae91wd_0lgK9lOR1qqY1HuDeXqSEAUIGrfh1VcP2n95Zc07EY-Uh3XjJE4drtgusACEK5n3P3WtN9s0m0GomEGQzF5ZJczxLGpHBKMQ5VDhMksVKdBAsx9xHgSx84aUhKQViYilAL-8PRj-RZA9s_IpEymAh5R37dKzAO8Fqq0nG7fVbH_ifzw3xhHiG92BhHldBDqEQ"),
	}

	tmpfile, errFs := os.CreateTemp("", "jwt")
	if errFs != nil {
		log.Fatal(errFs)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, errFs := tmpfile.Write(sts.jwt); errFs != nil {
		log.Fatal(errFs)
	}
	if errFs := tmpfile.Close(); errFs != nil {
		log.Fatal(errFs)
	}

	stsServer := httptest.NewServer(sts)
	defer stsServer.Close()
	os.Setenv("MC_STS_ENDPOINT", stsServer.URL+sts.endpoint)
	os.Setenv("MC_WEB_IDENTITY_TOKEN_FILE", tmpfile.Name())
	handler := adminPolicyHandler{
		endpoint: "/minio/admin/v3/add-canned-policy?name=",
		name:     "test",
		policy: []byte(`
{
  "Version": "2012-10-17",
  "Statement": [
	{
	  "Effect": "Allow",
	  "Action": [
		"s3:*"
	  ],
	  "Resource": [
		"arn:aws:s3:::test-bucket",
		"arn:aws:s3:::test-bucket/*"
	  ]
	}
  ]

}`),
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	conf := new(Config)
	conf.Debug = true
	conf.Insecure = true
	conf.HostURL = server.URL + handler.endpoint + handler.name
	s3c, err := s3AdminNew(conf)
	c.Assert(err, checkv1.IsNil)

	policyErr := s3c.AddCannedPolicy(context.Background(), handler.name, handler.policy)
	c.Assert(policyErr, checkv1.IsNil)
}
