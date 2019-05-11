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
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/pkg/grub"
	"github.com/urfave/cli"
)

var installCommand = cli.Command{
	Name:   "install",
	Usage:  "install terra onto a block device",
	Before: before,
	After:  after,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "http",
			Usage: "download via http",
		},
		cli.StringFlag{
			Name:  "device,d",
			Usage: "select the device terra is installed to",
			Value: "/dev/sda1",
		},
		cli.StringFlag{
			Name:  "fs-type",
			Usage: "set the filesystem type",
			Value: "ext4",
		},
		cli.StringFlag{
			Name:  "boot",
			Usage: "boot device to install the bootloader on",
		},
	},
	Action: func(clix *cli.Context) error {
		repo, err := getRepo(clix)
		if err != nil {
			return err
		}
		version := repo.Version()
		logger := logrus.WithField("repo", repo)
		logger.Info("setup directories")
		if err := setupDirectories(); err != nil {
			return errors.Wrap(err, "setup directories")
		}
		unpackTo := disk()
		if clix.String("fs-type") == "btrfs" {
			if err := createSubvolumes(version); err != nil {
				return errors.Wrap(err, "create subvolumes")
			}
			unpackTo = disk("submount")
			paths, err := mountSubvolumes(clix.String("device"), version, unpackTo)
			if err != nil {
				return errors.Wrap(err, "mount subvolumes")
			}
			defer func() {
				for _, p := range paths {
					syscall.Unmount(p, 0)
				}
			}()
		}

		logger.Info("creating new content store")
		store, err := newContentStore(disk("terra-content"))
		if err != nil {
			return errors.Wrap(err, "new content store")
		}
		defer os.RemoveAll(disk("terra-content"))

		ctx := cancelContext()
		desc, err := fetch(ctx, clix.Bool("http"), store, string(repo))
		if err != nil {
			return errors.Wrap(err, "fetch OS image")
		}
		if err := unpack(ctx, store, desc, unpackTo); err != nil {
			return err
		}
		if boot := clix.String("boot"); boot != "" {
			closer, err := overlayBoot()
			if err != nil {
				return err
			}
			defer closer()

			logger.Info("installing grub")
			if err := grub.Install(boot); err != nil {
				return err
			}
			logger.Info("making grub config")
			if err := grub.MkConfig(clix.String("device"), "/boot/grub/grub.cfg"); err != nil {
				return err
			}
		}
		return nil
	},
}
