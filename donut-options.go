package main

import (
	"github.com/minio-io/cli"
)

var healCmd = cli.Command{
	Name:        "heal",
	Usage:       "heal donut",
	Description: "",
	Action:      doHealCmd,
}

var attachCmd = cli.Command{
	Name:        "attach",
	Usage:       "attach disk",
	Description: "",
	Action:      doAttachCmd,
}

var detachCmd = cli.Command{
	Name:        "detach",
	Usage:       "detach disk",
	Description: "",
	Action:      doDetachCmd,
}

var rebalanceCmd = cli.Command{
	Name:        "rebalance",
	Usage:       "rebalance ",
	Description: "",
	Action:      doRebalanceCmd,
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
	healCmd,
	attachCmd,
	detachCmd,
	rebalanceCmd,
	cpDonutCmd,
	mbDonutCmd,
}

func doHealCmd(c *cli.Context) {
}

func doAttachCmd(c *cli.Context) {
}

func doDetachCmd(c *cli.Context) {
}

func doRebalanceCmd(c *cli.Context) {
}
