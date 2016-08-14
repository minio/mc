/*
 * Minio Client (C) 2016 Minio, Inc.
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

package command

import (
	"fmt"
	"sync"

	"github.com/minio/cli"
)

var (
	notifyFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "account-region",
			Value: "us-east-1",
			Usage: "Specify notification region. Defaults to ‘us-east-1’.",
		},
		cli.StringFlag{
			Name:  "account-id",
			Value: "mc",
			Usage: "Specify notification account id. Defaults to ‘mc’.",
		},
	}
)

var notifyCmd = cli.Command{
	Name:   "notify",
	Usage:  "Print realtime bucket notification.",
	Action: mainNotify,
	Flags:  append(notifyFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Watch new S3 operations on a minio server
      $ mc {{.Name}} myminio/testbucket

   2. Watch new events on a specific region and account id
      $ mc {{.Name}} myminio/testbucket --account-region us-west-2 --account-id 81132344
`,
}

// checkNotifySyntax - validate all the passed arguments
func checkNotifySyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "notify", 1) // last argument is exit code
	}
}

func mainNotify(ctx *cli.Context) {

	setGlobalsFromContext(ctx)

	checkNotifySyntax(ctx)

	args := ctx.Args()
	path := args[0]

	region := ctx.String("account-region")
	accountID := ctx.String("account-id")

	s3Client, pErr := newClient(path)
	if pErr != nil {
		fatalIf(pErr.Trace(), "Cannot parse the provided url.")
	}

	// Start watching on events
	wo, err := s3Client.Watch(watchParams{accountRegion: region, accountID: accountID, recursive: false})

	fatalIf(err, "Cannot watch on the specified bucket.")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event, ok := <-wo.Events():
				if !ok {
					return
				}
				fmt.Printf("%s\t%s\n", event.Type, event.Path)
			case err, ok := <-wo.Errors():
				if !ok {
					return
				}
				fmt.Printf("Error received: ", err)
			}
		}
	}()

	wg.Wait()
}
