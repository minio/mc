package main

import (
	"github.com/codegangsta/cli"
)

var format = cli.Command{
	Name:        "format",
	Usage:       "format disk",
	Description: "format a given disk",
	Action:      doFormat,
}

var heal = cli.Command{
	Name:        "heal",
	Usage:       "heal donut",
	Description: "",
	Action:      doHeal,
}

var attach = cli.Command{
	Name:        "attach",
	Usage:       "attach disk",
	Description: "",
	Action:      doAttach,
}

var detach = cli.Command{
	Name:        "detach",
	Usage:       "detach disk",
	Description: "",
	Action:      doDetach,
}

var rebalance = cli.Command{
	Name:        "rebalance",
	Usage:       "rebalance ",
	Description: "",
	Action:      doRebalance,
}

var donutOptions = []cli.Command{
	format,
	heal,
	attach,
	detach,
	rebalance,
}

func doFormat(c *cli.Context) {
}

func doHeal(c *cli.Context) {
}

func doAttach(c *cli.Context) {
}

func doDetach(c *cli.Context) {
}

func doRebalance(c *cli.Context) {
}
