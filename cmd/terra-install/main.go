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
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "terra-install"
	app.Version = version.Version
	app.Usage = "[repo] [destination]"
	app.Description = `
                                                     ___
                                                  ,o88888
                                               ,o8888888'
                         ,:o:o:oooo.        ,8O88Pd8888"
                     ,.::.::o:ooooOoOoO. ,oO8O8Pd888'"
                   ,.:.::o:ooOoOoOO8O8OOo.8OOPd8O8O"
                  , ..:.::o:ooOoOOOO8OOOOo.FdO8O8"
                 , ..:.::o:ooOoOO8O888O8O,COCOO"
                , . ..:.::o:ooOoOOOO8OOOOCOCO"
                 . ..:.::o:ooOoOoOO8O8OCCCC"o
                    . ..:.::o:ooooOoCoCCC"o:o
                    . ..:.::o:o:,cooooCo"oo:o:
                 ` + "`" + `   . . ..:.:cocoooo"'o:o:::'
                 .` + "`" + `   . ..::ccccoc"'o:o:o:::'
                :.:.    ,c:cccc"':.:.:.:.:.'
              ..:.:"'` + "`" + `::::c:"'..:.:.:.:.:.'
            ...:.'.:.::::"'    . . . . .'
           .. . ....:."' ` + "`" + `   .  . . ''
         . . . ...."'
         .. . ."'
        .
Terra OS management`
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.StringFlag{
			Name:  "device",
			Usage: "device name",
			Value: "/dev/sda1",
		},
		cli.StringFlag{
			Name:  "fs-type",
			Usage: "set the filesystem type",
			Value: "ext4",
		},
	}
	app.Before = func(clix *cli.Context) error {
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Action = func(clix *cli.Context) error {
		repo, err := image.GetRepo(clix)
		if err != nil {
			return err
		}
		var (
			version  = repo.Version()
			logger   = logrus.WithField("repo", repo)
			unpackTo = "/sd"
		)
		if err := os.MkdirAll("/sd", 0755); err != nil {
			return errors.Wrap(err, "create /sd")
		}
		if err := syscall.Mount(clix.GlobalString("device"), unpackTo, clix.GlobalString("fs-type"), 0, ""); err != nil {
			return errors.Wrap(err, "mount device")
		}
		defer syscall.Unmount("/sd", 0)

		if clix.GlobalString("fs-type") == "btrfs" {
			unpackTo = filepath.Join(unpackTo, "submount")
			if err := os.MkdirAll(unpackTo, 0755); err != nil {
				return errors.Wrap(err, "mkdir submount")
			}
			if err := btrfs.CreateSubvolumes(version, "/sd"); err != nil {
				return errors.Wrap(err, "create subvolumes")
			}
			paths, err := btrfs.MountSubvolumes(clix.String("device"), version, unpackTo)
			if err != nil {
				return errors.Wrap(err, "mount subvolumes")
			}
			defer func() {
				for _, p := range paths {
					syscall.Unmount(p, 0)
				}
			}()
			// write the version
			f, err := os.Create(filepath.Join(unpackTo, "VERSION"))
			if err != nil {
				return errors.Wrap(err, "create VERSION file")
			}
			_, err = fmt.Fprint(f, version)
			f.Close()
			if err != nil {
				return errors.Wrap(err, "write version")
			}
		}

		logger.Info("creating new content store")
		storePath := filepath.Join("/sd", "terra-content")
		store, err := image.NewContentStore(storePath)
		if err != nil {
			return errors.Wrap(err, "new content store")
		}

		ctx := cmd.CancelContext()
		desc, err := image.Fetch(ctx, clix.Bool("http"), store, string(repo))
		if err != nil {
			return errors.Wrap(err, "fetch OS image")
		}
		return image.Unpack(ctx, store, desc, unpackTo)
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var subvolumes = map[string]string{
	"home":       "/home",
	"containerd": "/var/lib/containerd",
	"log":        "/var/log",
}
