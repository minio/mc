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

var makeDonutCmd = cli.Command{
	Name:        "make",
	Usage:       "make",
	Description: "",
	Action:      doMakeDonutCmd,
}

var attachDiskCmd = cli.Command{
	Name:        "attach",
	Usage:       "attach disk",
	Description: "",
	Action:      doAttachDiskCmd,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "Donut name",
		},
	},
}

var detachDiskCmd = cli.Command{
	Name:        "detach",
	Usage:       "detach disk",
	Description: "",
	Action:      doDetachDiskCmd,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "Donut name",
		},
	},
}

var healDonutCmd = cli.Command{
	Name:        "heal",
	Usage:       "heal donut",
	Description: "",
	Action:      doHealDonutCmd,
}

var rebalanceDonutCmd = cli.Command{
	Name:        "rebalance",
	Usage:       "rebalance ",
	Description: "",
	Action:      doRebalanceDonutCmd,
}

var cpDonutCmd = cli.Command{
	Name:        "cp",
	Usage:       "cp",
	Description: "",
	Action:      doDonutCPCmd,
}

var mbDonutCmd = cli.Command{
	Name:        "mb",
	Usage:       "mb",
	Description: "",
	Action:      doMakeDonutBucketCmd,
}

var donutOptions = []cli.Command{
	makeDonutCmd,
	attachDiskCmd,
	detachDiskCmd,
	healDonutCmd,
	rebalanceDonutCmd,
	mbDonutCmd,
	cpDonutCmd,
}

func doHealDonutCmd(c *cli.Context) {
}

func doRebalanceDonutCmd(c *cli.Context) {
}
