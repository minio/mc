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
	"fmt"
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/auth"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminBucketReplicationSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "path-style",
		Value: "auto",
		Usage: "path style supported by the server. Valid options are '[on,off,auto]'",
	},
	cli.StringFlag{
		Name:  "api",
		Usage: "API signature. Valid options are '[S3v4, S3v2]'",
	},
}
var adminBucketReplicationSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set a new replication target",
	Action: mainAdminBucketReplicationSet,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, adminBucketReplicationSetFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET http(s)://ACCESSKEY:SECRETKEY@REPLICA_URL/REPLICA_BUCKET [--path | --api]

TARGET:
   Also called as alias/sourcebucketname

REPLICA_BUCKET:
  Also called as replication target bucket.

REPLICA_URL:
  Also called as replication endpoint.

ACCESSKEY:
  Also called as username.

SECRETKEY:
  Also called as password.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set a new replication target replicabucket on https://minio2:9000 for bucket 'srcbucket' to MinIO server.
     {{.DisableHistory}}
     {{.Prompt}} {{.HelpName}} myminio/srcbucket https://foobar:foo12345@minio2:9000/replicabucket
     {{.EnableHistory}}
`,
}

// checkAdminBucketReplicationSetSyntax - validate all the passed arguments
func checkAdminBucketReplicationSetSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 2 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
	if argsNr > 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for replication set command.")
	}

}

// replicationMessage container for content message structure
type replicationMessage struct {
	op             string
	Status         string `json:"status"`
	AccessKey      string `json:"accessKey,omitempty"`
	SecretKey      string `json:"secretKey,omitempty"`
	SourceBucket   string `json:"sourceBucket"`
	ReplicaURL     string `json:"replicaURL,omitempty"`
	ReplicaBucket  string `json:"replicaBucket,omitempty"`
	ReplicationARN string `json:"replicationARN,omitempty"`
	Path           string `json:"path,omitempty"`
	API            string `json:"api,omitempty"`
}

func (r replicationMessage) String() string {
	switch r.op {
	case "get":
		return console.Colorize("ReplicationMessage", "Found Replication target `"+r.ReplicaURL+"` with accessKey: "+r.AccessKey+" for "+r.SourceBucket+" -> "+r.ReplicaBucket+"\n Replication ARN = "+r.ReplicationARN)
	case "remove":
		return console.Colorize("ReplicationMessage", "Removed replication target for `"+r.SourceBucket+"` bucket successfully.")
	case "set":
		return console.Colorize("ReplicationMessage", "Replication ARN = `"+r.ReplicationARN+"`.")
	}
	return ""
}

func (r replicationMessage) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// isValidPath - validates if bucket path is of valid type
func isValidPath(path string) (ok bool) {
	l := strings.ToLower(strings.TrimSpace(path))
	for _, v := range []string{"on", "off", "auto"} {
		if l == v {
			return true
		}
	}
	return false
}

// fetchReplicationTarget - returns the dest bucket, dest endpoint, access and secret key
func fetchReplicationTarget(cli *cli.Context) (sourceBucket string, replTarget *madmin.BucketReplicationTarget, err error) {
	args := cli.Args()
	argCount := len(args)
	if argCount < 2 {
		return sourceBucket, replTarget, fmt.Errorf("Missing replication target configuration")
	}
	_, sourceBucket = url2Alias(args[0])
	replicaURL := args[1]
	api := cli.String("api")
	if api != "" && !isValidAPI(api) { // Empty value set to default "S3v4".
		fatalIf(errInvalidArgument().Trace(api),
			"Unrecognized API signature. Valid options are `[S3v4, S3v2]`.")
	}
	path := cli.String("path-style")
	if !isValidPath(path) {
		fatalIf(errInvalidArgument().Trace(path),
			"Unrecognized bucket path style. Valid options are `[on,off, auto]`.")
	}
	u, cerr := url.Parse(replicaURL)
	if cerr != nil {
		fatalIf(probe.NewError(cerr), "Malformed replication target URL")
	}
	isSSL := u.Scheme == "https"
	accessKey := u.User.Username()
	secretKey, _ := u.User.Password()
	replicaBucket := strings.TrimPrefix(u.Path, slashSeperator)
	replicaBucket = strings.TrimSuffix(replicaBucket, slashSeperator)

	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	creds := &auth.Credentials{AccessKey: accessKey, SecretKey: secretKey}
	replTarget = &madmin.BucketReplicationTarget{TargetBucket: replicaBucket, IsSSL: isSSL, Credentials: creds, Endpoint: u.Host, Path: path, API: api}
	return sourceBucket, replTarget, nil
}

// mainAdminBucketReplicationSet is the handle for "mc admin bucket replication set" command.
func mainAdminBucketReplicationSet(ctx *cli.Context) error {
	checkAdminBucketReplicationSetSyntax(ctx)

	console.SetColor("ReplicationMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	sourceBucket, replTarget, perr := fetchReplicationTarget(ctx)
	fatalIf(probe.NewError(perr), "Unable to parse input arguments.")
	fatalIf(probe.NewError(client.SetBucketReplicationTarget(globalContext, sourceBucket, replTarget)).Trace(args...), "Cannot add new replication target")
	arn, e := client.GetBucketReplicationARN(globalContext, replTarget.URL())
	fatalIf(probe.NewError(e), "Replication ARN missing")

	printMsg(replicationMessage{
		op:             "set",
		ReplicaURL:     replTarget.URL(),
		ReplicaBucket:  replTarget.TargetBucket,
		AccessKey:      replTarget.Credentials.AccessKey,
		SourceBucket:   sourceBucket,
		ReplicationARN: arn,
	})

	return nil
}
