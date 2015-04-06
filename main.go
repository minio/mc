/*
 * Minimalist Object Storage, (C) 2014, 2015 Minio, Inc.
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
	"os"
	"os/user"
	"runtime"
	"strconv"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/iodine"
)

func init() {
	// Check for the environment early on and gracefuly report.
	_, err := user.Current()
	if err != nil {
		msg := fmt.Sprintf("mc: Unable to obtain user's home directory %s\n", iodine.New(err, nil))
		fatal(msg)
	}

	// Ensures config file is sane and cached to _config private variable.
	config, err := getMcConfig()
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		msg := fmt.Sprintf("Unable to read config %s\n", iodine.New(err, nil))
		fatal(msg)
	}

	err = checkMcConfig(config)
	if err != nil {
		msg := fmt.Sprintf("Error in config file: %s\n%s", getMcConfigFilename(), iodine.New(err, nil))
		fatal(msg)
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
	app.Usage = "Minio Client for S3 Compatible Object Storage"
	app.Version = mcGitCommitHash
	app.Commands = options
	app.Flags = flags
	app.Author = "Minio.io"
	app.EnableBashCompletion = true
	app.Before = func(c *cli.Context) error {
		globalQuietFlag = c.GlobalBool("quiet")
		globalDebugFlag = c.GlobalBool("debug")
		if globalDebugFlag {
			app.ExtraInfo = getSystemData()
		}
		return nil
	}
	app.RunAndExitOnError()
}
