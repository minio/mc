/*
 * Mini Object Storage, (C) 2014 Minio, Inc.
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
	"github.com/codegangsta/cli"
)

var Cp = cli.Command{
	Name:        "cp",
	Usage:       "",
	Description: "",
	Action:      doCopy,
}

var Ls = cli.Command{
	Name:        "ls",
	Usage:       "",
	Description: "",
	Action:      doList,
}

var Mb = cli.Command{
	Name:        "mb",
	Usage:       "",
	Description: "",
	Action:      doMakeBucket,
}

var Mv = cli.Command{
	Name:        "mv",
	Usage:       "",
	Description: "",
	Action:      doMoveObject,
}

var Rb = cli.Command{
	Name:        "rb",
	Usage:       "",
	Description: "",
	Action:      doRemoveBucket,
}

var Rm = cli.Command{
	Name:        "rm",
	Usage:       "",
	Description: "",
	Action:      doRemoveObject,
}

var Sync = cli.Command{
	Name:        "sync",
	Usage:       "",
	Description: "",
	Action:      doSync,
}

func doCopy(c *cli.Context) {
}

func doList(c *cli.Context) {
}

func doMakeBucket(c *cli.Context) {
}

func doMoveObject(c *cli.Context) {
}

func doRemoveBucket(c *cli.Context) {
}

func doRemoveObject(c *cli.Context) {
}

func doSync(c *cli.Context) {
}
