/*
 * MinIO Client (C) 2020 MinIO, Inc.
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

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fatih/color"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
)

var bucketReplicateCmd = cli.Command{
	Name:   "replicate",
	Usage:  "configure bucket replication",
	Action: mainReplicate,
	Before: setGlobalsFromContext,
	Flags:  append(replicateFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}
 
 USAGE:
   {{.HelpName}} [COMMAND FLAGS] TARGET
 
 FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
 DESCRIPTION:
   Manage replication configuration on the bucket.
 
 EXAMPLES:
   1. Add a replication configuration to the "devices" bucket
      {{.Prompt}} {{.HelpName}} myminio/devices --config "config.json"

   2. List replication configuration on the "devices" bucket
      {{.Prompt}} {{.HelpName}} myminio/devices

   3. Clear replication configuration on the "devices" bucket
      {{.Prompt}} {{.HelpName}} myminio/devices --clear
`,
}

var replicateFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "config",
		Usage: "add a config.json with replication configuration",
	},
	cli.BoolFlag{
		Name:  "clear",
		Usage: "remove replication configuration",
	},
}

type replicateMessage struct {
	Op                string             `json:"-"`
	Status            string             `json:"status"`
	Source            string             `json:"source"`
	ReplicationConfig replication.Config `json:"replicationConfig,omitempty"`
}

func (r replicateMessage) String() string {
	switch r.Op {
	case "Add":
		return console.Colorize("ReplicateMessage", "Replication configuration added successfully to "+r.Source+".")
	case "Remove":
		return console.Colorize("ReplicateMessage", "Replication configuration removed successfully from "+r.Source+".")
	default:
		if r.ReplicationConfig.Empty() {
			return console.Colorize("ReplicateNMessage", "No replication configuration found for "+r.Source+".")
		}
		msgBytes, e := json.MarshalIndent(r.ReplicationConfig, "", " ")
		fatalIf(probe.NewError(e), "Unable to marshal replication configuration")
		return string(msgBytes)
	}
}

func (r replicateMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate user given arguments
func checkReplicateSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "replicate", globalErrorExitStatus)
	}

	if ctx.IsSet("config") && ctx.String("config") == "" {
		fatalIf(errInvalidArgument(), "config cannot be empty, please refer mc "+ctx.Command.FullName()+" --help for more details")
	}
}

// getReplicateArgs - returns the replication config and replication ARN
func getReplicateArgs(cli *cli.Context) (replConfig replication.Config, err error) {
	if !cli.IsSet("config") {
		return replConfig, nil
	}
	if cli.IsSet("clear") {
		return replConfig, fmt.Errorf("clear flag must be passed with target alone")
	}
	cfgFile := cli.String("config")
	fileReader, e := os.Open(cfgFile)
	if e != nil {
		return replConfig, fmt.Errorf("Unable to open config `" + cfgFile + "`.")
	}
	defer fileReader.Close()

	const maxJSONSize = 120 * 1024 // 120KiB
	configBuf, err := ioutil.ReadAll(fileReader)
	if err != nil {
		return replConfig, err
	}
	if len(configBuf) > maxJSONSize {
		return replConfig, bytes.ErrTooLarge
	}

	e = json.Unmarshal(configBuf, &replConfig)
	if e != nil {
		return replConfig, e
	}
	return
}

func mainReplicate(cliCtx *cli.Context) error {
	ctx, cancelReplicate := context.WithCancel(globalContext)
	defer cancelReplicate()
	checkReplicateSyntax(cliCtx)
	console.SetColor("ReplicateMessage", color.New(color.FgGreen))
	console.SetColor("ReplicateNMessage", color.New(color.FgRed))

	args := cliCtx.Args()
	urlStr := args.Get(0)
	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)
	replConfig, cerr := getReplicateArgs(cliCtx)
	fatalIf(probe.NewError(cerr), "Unable to parse input arguments.")
	var op string
	if !replConfig.Empty() {
		op = "Add"
		// Configuration that is already set.
		err = client.SetReplication(ctx, &replConfig)
		fatalIf(err.Trace(args...), "Unable to set replication configuration for "+urlStr)

	} else if cliCtx.IsSet("clear") {
		op = "Remove"
		err = client.RemoveReplication(ctx)
		fatalIf(err.Trace(args...), "Unable to remove replication configuration for "+urlStr)
	} else {
		replConfig, err = client.GetReplication(ctx)
		fatalIf(err.Trace(args...), "Unable to get replication configuration for "+urlStr)
	}
	printMsg(replicateMessage{
		Op:                op,
		Status:            "success",
		Source:            urlStr,
		ReplicationConfig: replConfig,
	})
	return nil
}
