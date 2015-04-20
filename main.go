/*
 * Mini Copy, (C) 2014, 2015 Minio, Inc.
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
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"runtime"
	"strconv"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

func checkConfig() {
	// Check for the environment early on and gracefuly report.
	_, err := user.Current()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to obtain user's home directory")
	}
	// Ensures config file is sane
	_, err = getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: Unable to read config file: %s\n", mustGetMcConfigPath())
	}
}

// Tries to get os/arch/platform specific information
// Returns a map of current os/arch/platform/memstats
func getSystemData() map[string]string {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)
	mem := fmt.Sprintf("Used: %s | Allocated: %s | Used-Heap: %s | Allocated-Heap: %s",
		pb.FormatBytes(int64(memstats.Alloc)),
		pb.FormatBytes(int64(memstats.TotalAlloc)),
		pb.FormatBytes(int64(memstats.HeapAlloc)),
		pb.FormatBytes(int64(memstats.HeapSys)))
	platform := fmt.Sprintf("Host: %s | OS: %s | Arch: %s",
		host,
		runtime.GOOS,
		runtime.GOARCH)
	goruntime := fmt.Sprintf("Version: %s | CPUs: %s", runtime.Version(), strconv.Itoa(runtime.NumCPU()))
	return map[string]string{
		"PLATFORM": platform,
		"RUNTIME":  goruntime,
		"MEM":      mem,
	}
}

func main() {
	app := cli.NewApp()
	app.Usage = "Mini Copy for object storage and filesystems"
	app.Version = mcGitCommitHash
	app.Commands = options
	app.Flags = flags
	app.Author = "Minio.io"
	app.Before = func(ctx *cli.Context) error {
		globalQuietFlag = ctx.GlobalBool("quiet")
		globalDebugFlag = ctx.GlobalBool("debug")
		if globalDebugFlag {
			app.ExtraInfo = getSystemData()
		} else {
			log.Debug = log.New(ioutil.Discard, "", 0)
		}

		theme := ctx.GlobalString("theme")
		if theme != "" {
			err := console.SetTheme(theme)
			if err != nil {
				console.Fatalf("mc: Unable to set theme [%s]\n", theme)
			}
		}
		checkConfig()
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		if !isMcConfigExist() && ctx.Command.Name != "config" {
			console.Fatalln("mc: error ``mc`` is not configured. Please run \"mc config generate\".")
		}

		return nil
	}
	app.RunAndExitOnError()
}
