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
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
	"golang.org/x/sys/unix"
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
			Name:  "os",
			Usage: "os device (include parition)",
		},
		cli.StringFlag{
			Name:  "data",
			Usage: "data device (include parition)",
		},
		cli.StringFlag{
			Name:  "version",
			Usage: "terraos version",
			Value: version.Version,
		},
	},
	Action: func(clix *cli.Context) error {
		var (
			ctx     = cmd.CancelContext()
			ver     = clix.String("version")
			osLun   = clix.String("os")
			dataLun = clix.String("data")
		)
		if ver == "" {
			return errors.New("no terraos version specified")
		}
		if osLun == "" {
			return errors.New("no os device specified")
		}

		// handle running on a provisioning machine vs in the iso
		storePath := filepath.Join("/tmp", "contentstore")
		if _, err := os.Stat(contentStorePath); err == nil {
			storePath = contentStorePath
		}
		devs := []dev{
			{
				label:  "os",
				device: osLun,
				path:   "/",
			},
		}
		if dataLun != "" {
			devs = append(devs, dev{
				label:  "data",
				device: dataLun,
				path:   "/var/lib",
			})
		}

		store, err := image.NewContentStore(storePath)
		if err != nil {
			return errors.Wrap(err, "new content store")
		}
		desc, err := image.Fetch(ctx, clix.Bool("http"), store, getTerraImage(clix, ver))
		if err != nil {
			return errors.Wrap(err, "fetch image")
		}
		dest := "/tmp/install"
		if err := os.MkdirAll(dest, 0755); err != nil {
			return errors.Wrapf(err, "mkdir for install")
		}
		for _, d := range devs {
			if err := d.mkfs(); err != nil {
				return errors.Wrapf(err, "format device %s", d)
			}
			// mount the entire diskmount group before subsystems
			closer, err := d.mount(dest)
			if err != nil {
				return err
			}
			defer closer()
		}
		// unpack image onto the destination
		if err := image.Unpack(ctx, store, desc, dest); err != nil {
			return errors.Wrap(err, "unpack image")
		}
		return nil
	},
}

type dev struct {
	label  string
	device string
	path   string
}

func (d *dev) String() string {
	return fmt.Sprintf("%s:%s", d.label, d.device)
}

func (d *dev) mkfs() error {
	return mkfs.Mkfs("ext4", d.label, d.device)
}

func (d *dev) mount(dest string) (func() error, error) {
	p := filepath.Join(dest, d.path)
	if err := os.MkdirAll(p, 0755); err != nil {
		return nil, errors.Wrapf(err, "mkdir %s", p)
	}
	if err := unix.Mount(d.device, p, "ext4", 0, ""); err != nil {
		return nil, errors.Wrapf(err, "mount %s to %s", d.label, p)
	}
	return func() error {
		return unix.Unmount(p, 0)
	}, nil
}

func getTerraImage(clix *cli.Context, version string) string {
	return filepath.Join(clix.GlobalString("repository"), fmt.Sprintf("%s:%s", terraImage, version))
}
