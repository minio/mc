/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
	"github.com/minio-io/cli"
)

var cpDonutCmd = cli.Command{
	Name:        "cp",
	Usage:       "cp",
	Description: "Copies a local file or dir or object or bucket to another location locally or to Donut or to S3.",
	Action:      doDonutCPCmd,
}

var mbDonutCmd = cli.Command{
	Name:        "mb",
	Usage:       "make bucket",
	Description: "Make a new bucket",
	Action:      doMakeDonutBucketCmd,
}

var lsDonutCmd = cli.Command{
	Name:        "ls",
	Usage:       "get list of objects",
	Description: `List Objects and common prefixes under a prefix or all Buckets`,
	Action:      doDonutListCmd,
}

var donutOptions = []cli.Command{
	mbDonutCmd,
	lsDonutCmd,
	cpDonutCmd,
}
