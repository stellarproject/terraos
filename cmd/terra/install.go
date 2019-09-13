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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/syslinux"
	"github.com/urfave/cli"
)

var installCommand = cli.Command{
	Name:  "install",
	Usage: "install terra onto a physical disk",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "http",
			Usage: "fetch image over http",
		},
		cli.StringSliceFlag{
			Name:  "device",
			Usage: "device for volume LABEL:dev",
			Value: &cli.StringSlice{},
		},
	},
	Action: func(clix *cli.Context) error {
		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		devices, err := getDevices(clix)
		if err != nil {
			return errors.Wrap(err, "get devices")
		}

		var (
			ctx     = cmd.CancelContext()
			scanner = bufio.NewScanner(os.Stdin)
		)

		// handle running on a provisioning machine vs in the iso
		storePath := filepath.Join("/tmp", "contentstore")
		if _, err := os.Stat(contentStorePath); err == nil {
			storePath = contentStorePath
		}

		store, err := image.NewContentStore(storePath)
		if err != nil {
			return errors.Wrap(err, "new content store")
		}
		desc, err := image.Fetch(ctx, clix.Bool("http"), store, node.Image.Name)
		if err != nil {
			return errors.Wrap(err, "fetch image")
		}

		dest := "/tmp/install"
		if err := os.MkdirAll(dest, 0755); err != nil {
			return errors.Wrapf(err, "mkdir for install")
		}
		var isISCSI bool
		for _, v := range node.Volumes {
			if !isISCSI {
				isISCSI = v.IsISCSI()
			}
			dev, ok := devices[v.Label]
			if !ok {
				if !v.IsISCSI() {
					return errors.Errorf("device for label %s does not exist", v.Label)
				}
				// mount the iscsi target if we have one
				if err := checkYes("login to iscsi device?", scanner); err != nil {
					return err
				}
				logrus.Info("mounting iscsi target")
				if dev, err = v.Login(ctx, node.Pxe.IscsiTarget); err != nil {
					return errors.Wrap(err, "login iscsi")
				}
				defer v.Logout(ctx, node.Pxe.IscsiTarget)
				if err := checkYes(fmt.Sprintf("installing terra to %s", dev), scanner); err != nil {
					return err
				}
			}
			if err := v.Format(dev); err != nil {
				return errors.Wrap(err, "format volume")
			}
			// mount the entire diskmount group before subsystems
			closer, err := v.Mount(dev, dest)
			if err != nil {
				return err
			}
			defer closer()
		}
		// unpack image onto the destination
		if err := image.Unpack(ctx, store, desc, dest); err != nil {
			return errors.Wrap(err, "unpack image")
		}
		if isISCSI {
			// for iscsi, there is no need to setup a boot device
			return nil
		}
		for _, v := range node.Volumes {
			dev, ok := devices[v.Label]
			if ok && v.Boot {
				logrus.Info("installing bootloader")
				path := dest
				if err := syslinux.Copy(path); err != nil {
					return errors.Wrap(err, "copy syslinux from live cd")
				}
				if err := syslinux.InstallMBR(removePartition(dev), "/boot/syslinux/mbr.bin"); err != nil {
					return errors.Wrap(err, "install mbr")
				}
				if err := syslinux.ExtlinuxInstall(filepath.Join(path, "boot", "syslinux")); err != nil {
					return errors.Wrap(err, "install extlinux")
				}
			}
		}
		return nil
	},
}

func removePartition(device string) string {
	partition := string(device[len(device)-1])
	if _, err := strconv.Atoi(partition); err != nil {
		return device
	}
	if strings.Contains(device, "nvme") {
		partition = "p" + partition
	}
	return strings.TrimSuffix(device, partition)
}

func getDevices(clix *cli.Context) (map[string]string, error) {
	var (
		out = make(map[string]string)
	)
	for _, d := range clix.StringSlice("device") {
		parts := strings.Split(d, ":")
		if len(parts) != 2 {
			return nil, errors.Errorf("device %s not valid format", d)
		}
		out[parts[0]] = parts[1]
	}
	return out, nil
}

func checkYes(msg string, scanner *bufio.Scanner) error {
	fmt.Fprint(os.Stderr, msg)
	fmt.Fprintln(os.Stderr, " [y/n]")
	if !scanner.Scan() {
		return errors.New("no input")
	}
	if strings.ToLower(scanner.Text()) == "y" {
		return nil
	}
	return errors.New("user aborted")
}
