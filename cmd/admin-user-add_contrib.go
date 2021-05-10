// MinIO Object Storage (c) 2021 MinIO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
	"golang.org/x/crypto/ssh/terminal"
)

// fetchUserKeys - returns the access and secret key
func fetchUserKeys(args cli.Args) (string, string) {
	accessKey := ""
	secretKey := ""
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	isTerminal := terminal.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	argCount := len(args)

	if argCount == 1 {
		if isTerminal {
			fmt.Printf("%s", console.Colorize(cred, "Enter Access Key: "))
		}
		value, _, _ := reader.ReadLine()
		accessKey = string(value)
	} else {
		accessKey = args.Get(1)
	}

	if argCount == 1 || argCount == 2 {
		if isTerminal {
			fmt.Printf("%s", console.Colorize(cred, "Enter Secret Key: "))
			bytePassword, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Printf("\n")
			secretKey = string(bytePassword)
		} else {
			value, _, _ := reader.ReadLine()
			secretKey = string(value)
		}
	} else {
		secretKey = args.Get(2)
	}

	return accessKey, secretKey
}
