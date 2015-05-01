/*
 * Minio Client (C) 2015 Minio, Inc.
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

package s3

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/s3"
	"github.com/minio-io/mc/pkg/client"
)

// New returns an initialized s3Client structure.
// if debug use a internal trace transport
func New(config *Config) client.Client {
	u, err := url.Parse(config.HostURL)
	if err != nil {
		return nil
	}
	var traceTransport RoundTripTrace
	var transport http.RoundTripper
	if config.Debug {
		traceTransport = GetNewTraceTransport(NewTrace(false, true, nil), http.DefaultTransport)
		transport = GetNewTraceTransport(s3Verify{}, traceTransport)
	} else {
		transport = http.DefaultTransport
	}
	awsConf := aws.DefaultConfig
	awsConf.Credentials = aws.Creds(config.AccessKeyID, config.SecretAccessKey, "")
	awsConf.HTTPClient = &http.Client{Transport: transport}
	awsConf.Logger = ioutil.Discard
	s3c := &s3Client{
		&Meta{
			Config:    config,
			Transport: transport,
			S3:        s3.New(awsConf),
		}, u,
	}
	return s3c
}
