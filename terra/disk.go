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
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var autoPartitionCommand = cli.Command{
	Name:  "auto-partition",
	Usage: "configure the os",
	Action: func(clix *cli.Context) error {
		if err := partition(clix); err != nil {
			return err
		}
		if err := format(clix); err != nil {
			return err
		}
		return installGrub(clix)
	},
}

func partition(clix *cli.Context) error {
	const parted = "o\nn\np\n1\n\n\na\n1\nw"

	cmd := exec.Command("fdisk", clix.GlobalString("device"))
	cmd.Stdin = strings.NewReader(parted)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func format(clix *cli.Context) error {
	cmd := exec.Command(fmt.Sprintf("mkfs.%s", clix.GlobalString("fs-type")), partitionPath(clix.GlobalString("device")))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installGrub(clix *cli.Context) error {
	if err := os.MkdirAll(devicePath, 0755); err != nil {
		return err
	}
	path := partitionPath(clix.GlobalString("device"))
	logrus.WithFields(logrus.Fields{
		"device": path,
		"path":   devicePath,
	}).Info("mounting device")
	if err := syscall.Mount(path, devicePath, clix.GlobalString("fs-type"), 0, ""); err != nil {
		return err
	}
	defer syscall.Unmount(devicePath, 0)

	if err := os.MkdirAll(disk("boot"), 0755); err != nil {
		return err
	}
	if err := syscall.Mount(disk("boot"), "/boot", "none", syscall.MS_BIND, ""); err != nil {
		return err
	}
	defer syscall.Unmount("/boot", 0)
	out, err := exec.Command("grub-install", clix.GlobalString("device")).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
