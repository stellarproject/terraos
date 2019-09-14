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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/types/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/version"
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
		pxeMoudlesCommand,
		pxeSaveCommand,
	},
}

var pxeMoudlesCommand = cli.Command{
	Name:        "modules",
	Description: "install a new pxe's modules to the system",
	ArgsUsage:   "[image]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "location,l",
			Usage: "modules location",
			Value: "/lib/modules",
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
		var (
			source = filepath.Join(path, "lib/modules") + "/"
			target = clix.String("location") + "/"
		)
		if err := syncDir(ctx, source, target); err != nil {
			return errors.Wrap(err, "sync tftp dir")
		}
		return nil
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
		cli.BoolFlag{
			Name:  "default",
			Usage: "set this pxe install as the default",
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
		var (
			source = filepath.Join(path, "tftp") + "/"
			target = clix.String("tftp") + "/"
		)
		if clix.Bool("default") {
			if err := syncDir(ctx, source, target); err != nil {
				return errors.Wrap(err, "sync tftp dir")
			}
		} else {
			if err := copyKernel(source, getVersion(i), target); err != nil {
				return errors.Wrap(err, "copy kernel")
			}
		}
		return nil
	},
}

var pxeSaveCommand = cli.Command{
	Name:        "save",
	Description: "save a node's pxe configuration",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "tftp,t",
			Usage: "tftp location",
			Value: "/tftp",
		},
		cli.StringFlag{
			Name:  "kv",
			Usage: "kernel version to pin for the config",
			Value: version.Version,
		},
	},
	Action: func(clix *cli.Context) error {
		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		path := filepath.Join(clix.String("tftp"), configDir, v1.PXEFilename(node))
		f, err := os.Create(path)
		if err != nil {
			return errors.Wrapf(err, "create pxe config file %s", path)
		}
		defer f.Close()

		if err := node.PXEConfig(f, clix.String("kv")); err != nil {
			return errors.Wrap(err, "write pxe configuration")
		}
		return nil
	},
}

const kvFmt = "%s-%s"

func copyKernel(source, version, target string) error {
	// rename kernel images
	for _, name := range []string{initrd, kernel} {
		sourceFile := filepath.Join(source, name)
		fn := filepath.Join(source, fmt.Sprintf(kvFmt, name, version))
		if err := os.Rename(sourceFile, fn); err != nil {
			return errors.Wrap(err, "rename kernels to target")
		}
		sf, err := os.Open(fn)
		if err != nil {
			return err
		}

		fn = filepath.Join(target, fmt.Sprintf(kvFmt, name, version))
		f, err := os.Create(fn)
		if err != nil {
			sf.Close()
			return err
		}
		_, err = io.Copy(f, sf)
		sf.Close()
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
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

func getVersion(i string) string {
	parts := strings.Split(i, ":")
	return parts[len(parts)-1]
}
