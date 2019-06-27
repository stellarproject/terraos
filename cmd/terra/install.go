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

	"github.com/containerd/containerd/content"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/types/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/resolvconf"
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
		cli.StringFlag{
			Name:  "gateway",
			Usage: "gateway ip",
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
		gateway := clix.String("gateway")
		if gateway == "" {
			return errors.New("--gateway not specified")
		}

		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		imageName := clix.Args().Get(1)
		if imageName == "" {
			return errors.New("no image passed on the command line")
		}
		devices, err := getDevices(clix)
		if err != nil {
			return errors.Wrap(err, "get devices")
		}
		var (
			diskmount = "/tmp/mnt"
			dest      = "/tmp/dest"
			ctx       = cmd.CancelContext()
		)
		for _, p := range []string{diskmount, dest} {
			if err := os.MkdirAll(p, 0755); err != nil {
				return errors.Wrapf(err, "mkdir %s", p)
			}
		}

		var store content.Store
		for _, v := range node.Volumes {
			dev, ok := devices[v.Label]
			if !ok {
				return errors.Errorf("device for label %s does not exist", v.Label)
			}
			if err := v.Format(dev.Device); err != nil {
				return errors.Wrap(err, "format volume")
			}
			path := filepath.Join(diskmount, v.Label)
			if err := os.MkdirAll(path, 0755); err != nil {
				return errors.Wrap(err, "mkdir volume mount")
			}
			// mount the entire diskmount group before subsystems
			closer, err := v.Mount(dev.Device, path)
			if err != nil {
				return err
			}
			defer closer()

			if store == nil {
				storePath := filepath.Join(path, "terra-install-content")
				if store, err = image.NewContentStore(storePath); err != nil {
					return errors.Wrap(err, "new content store")
				}
			}
		}
		if store == nil {
			return errors.New("store not created on any group")
		}
		// install
		desc, err := image.Fetch(ctx, clix.Bool("http"), store, imageName)
		if err != nil {
			return errors.Wrap(err, "fetch image")
		}
		if err := image.Unpack(ctx, store, desc, dest); err != nil {
			return errors.Wrap(err, "unpack image")
		}
		for _, v := range node.Volumes {
			dev, ok := devices[v.Label]
			if ok && dev.Boot {
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
		if err := writeResolvconf(dest, gateway); err != nil {
			return errors.Wrap(err, "write resolv.conf")
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

func writeFstab(entries []*fstab.Entry, root string) error {
	f, err := os.Create(filepath.Join(root, fstab.Path))
	if err != nil {
		return errors.Wrap(err, "create fstab file")
	}
	defer f.Close()
	if err := fstab.Write(f, entries); err != nil {
		return errors.Wrap(err, "write fstab")
	}
	return nil
}

func writeResolvconf(root, gateway string) error {
	path := filepath.Join(root, resolvconf.DefaultPath)
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "create resolv.conf file %s", path)
	}
	defer f.Close()

	conf := &resolvconf.Conf{
		Nameservers: []string{
			gateway,
		},
	}
	return conf.Write(f)
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
