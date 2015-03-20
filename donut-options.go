package main

import (
	"github.com/codegangsta/cli"
)

var formatCmd = cli.Command{
	Name:        "format",
	Usage:       "format disk",
	Description: "format a given disk",
	Action:      doFormatCmd,
}

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

var donutOptions = []cli.Command{
	formatCmd,
	healCmd,
	attachCmd,
	detachCmd,
	rebalanceCmd,
}

func doFormatCmd(c *cli.Context) {
}

func doHealCmd(c *cli.Context) {
}

func doAttachCmd(c *cli.Context) {
}

func doDetachCmd(c *cli.Context) {
}

func doRebalanceCmd(c *cli.Context) {
}
