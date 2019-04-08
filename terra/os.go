/*
	Copyright (c) 2019 Stellar Project

	Permission is hereby granted, free of charge, to any person
	obtaining a copy of this software and associated documentation
	files (the "Software"), to deal in the Software without
	restriction, including without limitation the rights to use, copy,
	modify, merge, publish, distribute, sublicense, and/or sell copies
	of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be
	included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
	EXPRESS OR IMPLIED,
	INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
	IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
	HOLDERS BE LIABLE FOR ANY CLAIM,
	DAMAGES OR OTHER LIABILITY,
	WHETHER IN AN ACTION OF CONTRACT,
	TORT OR OTHERWISE,
	ARISING FROM, OUT OF OR IN CONNECTION WITH
	THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"text/tabwriter"

	"github.com/urfave/cli"
)

var osCommand = cli.Command{
	Name:  "os",
	Usage: "manage the os's on your node",
	Subcommands: []cli.Command{
		downloadCommand,
		listCommand,
		enableCommand,
		configureCommand,
	},
}

var configureCommand = cli.Command{
	Name:   "config",
	Usage:  "configure the os",
	Before: before,
	After:  after,
	Action: func(clix *cli.Context) error {
		version, err := getVersion(clix)
		if err != nil {
			return err
		}
		if err := os.MkdirAll("/run/config", 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(disk("work"), 0755); err != nil {
			return err
		}
		if err := syscall.Mount("overlay", "/run/config", "overlay", 0,
			fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", disk(root, strconv.Itoa(version)), disk("config"), disk("work")),
		); err != nil {
			return err
		}
		defer syscall.Unmount("/run/config", 0)

		cmd := exec.Command("/bin/bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = "/"
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Chroot: "/run/config",
		}
		return cmd.Run()
	},
}

var enableCommand = cli.Command{
	Name:   "enable",
	Usage:  "enable a specific os",
	Before: before,
	After:  after,
	Action: func(clix *cli.Context) error {
		version, err := getVersion(clix)
		if err != nil {
			return err
		}
		return updateGrub(version)
	},
}

var listCommand = cli.Command{
	Name:   "list",
	Usage:  "list downloaded os versions",
	Before: before,
	After:  after,
	Action: func(clix *cli.Context) error {
		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		fmt.Fprint(w, "ID\tENABLED\n")
		dirs, err := ioutil.ReadDir(disk(root))
		if err != nil {
			if os.IsNotExist(err) {
				return w.Flush()
			}
			return err
		}
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			var (
				name = d.Name()
				e    = "*"
			)
			if name == "base" {
				continue
			}
			fmt.Fprintf(w, "%s\t%s\n", name, e)
		}
		return w.Flush()
	},
}

var downloadCommand = cli.Command{
	Name:      "download",
	Usage:     "download terra os",
	ArgsUsage: "VERSION",
	Before:    before,
	After:     after,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "http,i",
			Usage: "pull over http",
		},
	},
	Action: func(clix *cli.Context) (err error) {
		version, err := getVersion(clix)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(disk("config"), 0755); err != nil {
			return err
		}
		if err := applyImage(clix, fmt.Sprintf(bootRepoFormat, version), disk()); err != nil {
			return err
		}
		return applyImage(clix, fmt.Sprintf(terraRepoFormat, version), disk(root, strconv.Itoa(version)))
	},
}
