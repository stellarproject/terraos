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
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/pxe"
	"github.com/urfave/cli"
)

const (
	kernel    = "vmlinuz"
	initrd    = "initrd.img"
	configDir = "pxelinux.cfg"
)

var pxeCommand = cli.Command{
	Name:        "pxe",
	Description: "manage the pxe setup for terra",
	Subcommands: []cli.Command{
		pxeInstallCommand,
		pxeSaveCommand,
	},
}

var pxeInstallCommand = cli.Command{
	Name:        "install",
	Description: "install a new pxe image to a directory",
	ArgsUsage:   "[image]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "tftp,t",
			Usage: "tftp location",
			Value: "/tftp",
		},
		cli.BoolFlag{
			Name:  "http",
			Usage: "fetch over http",
		},
	},
	Action: func(clix *cli.Context) error {
		ctx := cmd.CancelContext()
		i := clix.Args().First()
		if i == "" {
			return errors.New("image config should be passed on command line")
		}
		store, err := getStore()
		if err != nil {
			return errors.Wrap(err, "get content store")
		}
		img, err := image.Fetch(ctx, clix.Bool("http"), store, i)
		if err != nil {
			return errors.Wrapf(err, "fetch %s", i)
		}
		path, err := ioutil.TempDir("", "terra-pxe-install")
		if err != nil {
			return errors.Wrap(err, "create tmp pxe dir")
		}
		defer os.RemoveAll(path)

		if err := image.Unpack(ctx, store, img, path); err != nil {
			return errors.Wrap(err, "unpack pxe image")
		}
		if err := syncDir(ctx, filepath.Join(path, "tftp")+"/", clix.String("tftp")+"/"); err != nil {
			return errors.Wrap(err, "sync tftp dir")
		}
		return nil
	},
}

var pxeSaveCommand = cli.Command{
	Name:        "save",
	Description: "save a node's pxe configuration",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "iscsi-target,target",
			Usage: "iscsi target IP",
		},
		cli.StringFlag{
			Name:  "tftp,t",
			Usage: "tftp location",
			Value: "/tftp",
		},
	},
	Action: func(clix *cli.Context) error {
		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		if len(node.Nics) == 0 {
			return errors.New("node must have atlest 1 NIC for PXE")
		}
		var (
			ip  = "dhcp"
			nic = node.Nics[0]
		)
		if len(nic.Addresses) > 0 {
			// FIXME: this will have to be fixed to get gateway, subnet, etc
			ip = nic.Addresses[0]
		}
		p := &pxe.PXE{
			Default: "pxe",
			MAC:     nic.Mac,
			IP:      ip,
			Entries: []pxe.Entry{
				{
					Root:   "LABEL=os",
					Boot:   "pxe",
					Label:  "pxe",
					Kernel: kernel,
					Initrd: initrd,
					// TODO: support options
				},
			},
		}
		for _, v := range node.Volumes {
			if v.IsISCSI() {
				p.TargetIP = clix.String("iscsi-target")
				p.TargetIQN = v.TargetIqn
				p.InitiatorIQN = node.IQN()
				break
			}
		}
		path := filepath.Join(clix.String("tftp"), configDir, p.Filename())
		f, err := os.Create(path)
		if err != nil {
			return errors.Wrapf(err, "create pxe config file %s", path)
		}
		defer f.Close()
		if err := p.Write(f); err != nil {
			return errors.Wrap(err, "write pxe configuration")
		}
		return nil
	},
}

func syncDir(ctx context.Context, source, target string) error {
	cmd := exec.CommandContext(ctx, "rsync", "--progress", "-a", source, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to rsync directories")
	}
	return nil
}
