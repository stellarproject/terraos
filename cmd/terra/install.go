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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/types/v1"
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
		cli.StringFlag{
			Name:  "boot",
			Usage: "select the boot device by label",
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

		ctx := cmd.CancelContext()
		store, err := image.NewContentStore(filepath.Join("/tmp", "contentstore"))
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

		for _, v := range node.Volumes {
			dev, ok := devices[v.Label]
			if !ok {
				return errors.Errorf("device for label %s does not exist", v.Label)
			}
			if err := v.Format(dev.Device); err != nil {
				return errors.Wrap(err, "format volume")
			}
			// mount the entire diskmount group before subsystems
			closer, err := v.Mount(dev.Device, dest)
			if err != nil {
				return err
			}
			defer closer()
		}
		// unpack image onto the destination
		if err := image.Unpack(ctx, store, desc, dest); err != nil {
			return errors.Wrap(err, "unpack image")
		}
		for _, v := range node.Volumes {
			dev, ok := devices[v.Label]
			if ok && dev.Boot {
				logrus.Info("installing bootloader")
				path := dest
				if err := syslinux.Copy(path); err != nil {
					return errors.Wrap(err, "copy syslinux from live cd")
				}
				if err := syslinux.InstallMBR(removePartition(dev.Device), "/boot/syslinux/mbr.bin"); err != nil {
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

func getDevices(clix *cli.Context) (map[string]v1.Disk, error) {
	var (
		boot = clix.String("boot")
		out  = make(map[string]v1.Disk)
	)
	for _, d := range clix.StringSlice("device") {
		parts := strings.Split(d, ":")
		if len(parts) != 2 {
			return nil, errors.Errorf("device %s not valid format", d)
		}
		out[parts[0]] = v1.Disk{
			Device: parts[1],
			Boot:   boot == parts[0],
		}
	}
	return out, nil
}
