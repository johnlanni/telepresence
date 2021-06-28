// Copyright 2021 Datawire. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
)

func Main() error {
	gitDescBytes, err := exec.Command("git", "describe", "--tags", "--match=v*").Output()
	if err != nil {
		return err
	}
	gitDescStr := strings.TrimSuffix(strings.TrimPrefix(string(gitDescBytes), "v"), "\n")
	gitDescVer, err := semver.Parse(gitDescStr)
	if err != nil {
		return err
	}
	gitDescVer.Patch++

	// If an additional arg has been used, we include it in the tag
	if len(os.Args) >= 2 {
		t := time.Now()
		date := fmt.Sprintf("%d.%d.%d", t.Year(), t.Month(), t.Day())
		_, err = fmt.Printf("v%d.%d.%d-%s-%s-%s\n", gitDescVer.Major, gitDescVer.Minor, gitDescVer.Patch, os.Args[1], date, gitDescVer.Pre[0].String())
	} else {
		_, err = fmt.Printf("v%s-%d\n", gitDescVer, time.Now().Unix())
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := Main(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", filepath.Base(os.Args[0]), err)
		os.Exit(1)
	}
}
