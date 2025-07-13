// Copyright (c) 2015-2023 MinIO, Inc.
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
	"log"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSTSS3Operation(t *testing.T) {
	sts := stsHandler{
		endpoint: "/",
		jwt:      []byte("eyJhbGciOiJSUzI1NiIsImtpZCI6Inc0dFNjMEc5Tk0wQWhGaWJYaWIzbkpRZkRKeDc1dURRTUVpOTNvTHJ0OWcifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzMxMTIyNjg0LCJpYXQiOjE2OTk1ODY2ODQsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJtaW5pby10ZW5hbnQtMSIsInBvZCI6eyJuYW1lIjoic2V0dXAtYnVja2V0LXJ4aHhiIiwidWlkIjoiNmNhMzhjMmItYTdkMC00M2Y0LWE0NjMtZjdlNjU4MGUyZDdiIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJtYy1qb2Itc2EiLCJ1aWQiOiI3OTc4NzJjZC1kMjkwLTRlM2EtYjYyMC00ZGFkYzZhNzUyMTYifSwid2FybmFmdGVyIjoxNjk5NTkwMjkxfSwibmJmIjoxNjk5NTg2Njg0LCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6bWluaW8tdGVuYW50LTE6bWMtam9iLXNhIn0.fBJckmoQFyJ9bUgKZv6jzBESd9ccX_HFPPBZ17Gz_CsQ5wXrMqnvoMs1mcv6QKWsDsvSnWnw_tcW0cjvVkXb2mKmioKLzqV4ihGbiWzwk2e1xDohn8fizdQkf64bXpncjGdEGv8oi9A4300jfLMfg53POriMyEAQMeIDKPOI9qx913xjGni2w2H49mjLfnFnRaj9osvy17425dNIrMC6GDFq3rcq6Z_cdDmL18Jwsjy1xDsAhUzmOclr-VI3AeSnuD4fbf6jhbKE14qVUjLmIBf__B5NhESiaFNwxFYjonZyi357Nx93CD1wai28tNRSODx7BiPHLxk8SyzY0CP0sQ"),
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
	t.Setenv("MC_STS_ENDPOINT_test", stsServer.URL+sts.endpoint)
	t.Setenv("MC_WEB_IDENTITY_TOKEN_FILE_test", tmpfile.Name())
	object := objectHandler{
		resource: "/bucket/object",
		data:     []byte("Hello, World"),
	}
	server := httptest.NewServer(object)
	defer server.Close()

	conf := new(Config)
	conf.Alias = "test"
	conf.HostURL = server.URL + object.resource
	s3c, err := S3New(conf)
	if err != nil {
		t.Fatal(err)
	}

	var reader io.Reader = bytes.NewReader(object.data)
	n, err := s3c.Put(context.Background(), reader, int64(len(object.data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(object.data)) {
		t.Fatalf("expected %d, got %d", n, len(object.data))
	}
}

func TestAdminSTSOperation(t *testing.T) {
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
	t.Setenv("MC_STS_ENDPOINT_test", stsServer.URL+sts.endpoint)
	t.Setenv("MC_WEB_IDENTITY_TOKEN_FILE_test", tmpfile.Name())
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
	conf.Alias = "test"
	conf.Debug = true
	conf.Insecure = true
	conf.HostURL = server.URL + handler.endpoint + handler.name
	s3c, err := s3AdminNew(conf)
	if err != nil {
		t.Fatal(err)
	}

	e := s3c.AddCannedPolicy(context.Background(), handler.name, handler.policy)
	if e != nil {
		t.Fatal(e)
	}
}
