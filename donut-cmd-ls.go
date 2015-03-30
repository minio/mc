/*
 * Minimalist Object Storage, (C) 2014,2015 Minio, Inc.
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

import (
	"fmt"

	"net/url"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client/donut"
)

// doDonutListCmd - list buckets and objects
func doDonutListCmd(c *cli.Context) {
	if !c.Args().Present() {
		fatal("no args?")
	}
	urlStr, err := parseURL(c.Args().First())
	if err != nil {
		fatal(err.Error())
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		fatal(err.Error())
	}
	mcDonutConfigData, err := loadDonutConfig()
	if err != nil {
		fatal(err.Error())
	}
	if _, ok := mcDonutConfigData.Donuts[u.Host]; !ok {
		msg := fmt.Sprintf("requested donut: <%s> does not exist", u.Host)
		fatal(msg)
	}
	nodes := make(map[string][]string)
	for k, v := range mcDonutConfigData.Donuts[u.Host].Node {
		nodes[k] = v.ActiveDisks
	}
	d, err := donut.GetNewClient(u.Host, nodes)
	if err != nil {
		fatal(err.Error())
	}
	buckets, err := d.ListBuckets()
	if err != nil {
		fatal(err.Error())
	}
	printBuckets(buckets)
}
