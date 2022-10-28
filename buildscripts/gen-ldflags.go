//go:build ignore
// +build ignore

// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func genLDFlags(version string) string {
	releaseTag, date := releaseTag(version)
	copyrightYear := fmt.Sprintf("%d", date.Year())

	var ldflagsStr string
	ldflagsStr = "-s -w -X github.com/minio/mc/cmd.Version=" + version + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.CopyrightYear=" + copyrightYear + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.ReleaseTag=" + releaseTag + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.CommitID=" + commitID() + " "
	ldflagsStr = ldflagsStr + "-X github.com/minio/mc/cmd.ShortCommitID=" + commitID()[:12]
	return ldflagsStr
}

// releaseTag prints release tag to the console for easy git tagging.
func releaseTag(version string) (string, time.Time) {
	relPrefix := "DEVELOPMENT"
	if prefix := os.Getenv("MC_RELEASE"); prefix != "" {
		relPrefix = prefix
	}

	relSuffix := ""
	if hotfix := os.Getenv("MC_HOTFIX"); hotfix != "" {
		relSuffix = hotfix
	}

	relTag := strings.ReplaceAll(version, " ", "-")
	relTag = strings.ReplaceAll(relTag, ":", "-")
	t, err := time.Parse("2006-01-02T15-04-05Z", relTag)
	if err != nil {
		panic(err)
	}

	relTag = strings.ReplaceAll(relTag, ",", "")
	relTag = relPrefix + "." + relTag
	if relSuffix != "" {
		relTag += "." + relSuffix
	}

	return relTag, t
}

// commitID returns the abbreviated commit-id hash of the last commit.
func commitID() string {
	// git log --format="%h" -n1
	var (
		commit []byte
		e      error
	)
	cmdName := "git"
	cmdArgs := []string{"log", "--format=%H", "-n1"}
	if commit, e = exec.Command(cmdName, cmdArgs...).Output(); e != nil {
		fmt.Fprintln(os.Stderr, "Error generating git commit-id: ", e)
		os.Exit(1)
	}

	return strings.TrimSpace(string(commit))
}

func commitTime() time.Time {
	// git log --format=%cD -n1
	var (
		commitUnix []byte
		err        error
	)
	cmdName := "git"
	cmdArgs := []string{"log", "--format=%cI", "-n1"}
	if commitUnix, err = exec.Command(cmdName, cmdArgs...).Output(); err != nil {
		fmt.Fprintln(os.Stderr, "Error generating git commit-time: ", err)
		os.Exit(1)
	}

	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(commitUnix)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating git commit-time: ", err)
		os.Exit(1)
	}

	return t.UTC()
}

func main() {
	var version string
	if len(os.Args) > 1 {
		version = os.Args[1]
	} else {
		version = commitTime().Format(time.RFC3339)
	}

	fmt.Println(genLDFlags(version))
}
