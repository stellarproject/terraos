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
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/cmd"
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
		repo, err := cmd.GetRepo(clix)
		if err != nil {
			return err
		}
		var (
			version  = repo.Version()
			logger   = logrus.WithField("repo", repo)
			unpackTo = clix.Args().Get(1)
		)
		if clix.GlobalString("fs-type") == "btrfs" {
			unpackTo = filepath.Join(unpackTo, "submount")
			if err := os.MkdirAll(unpackTo, 0755); err != nil {
				return errors.Wrap(err, "mkdir submount")
			}
			if err := createSubvolumes(version, clix.Args().Get(1)); err != nil {
				return errors.Wrap(err, "create subvolumes")
			}
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
		storePath := filepath.Join(clix.Args().Get(1), "terra-content")
		store, err := cmd.NewContentStore(storePath)
		if err != nil {
			return errors.Wrap(err, "new content store")
		}
		defer os.RemoveAll(storePath)

		ctx := cmd.CancelContext()
		desc, err := cmd.Fetch(ctx, clix.Bool("http"), store, string(repo))
		if err != nil {
			return errors.Wrap(err, "fetch OS image")
		}
		return cmd.Unpack(ctx, store, desc, unpackTo)
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

func createSubvolumes(version, path string) error {
	for d := range subvolumes {
		sv := filepath.Join(path, d)
		if err := btrfs("subvolume", "create", sv); err != nil {
			return err
		}
	}
	vp := filepath.Join(path, version)
	return btrfs("subvolume", "create", vp)
}

func mountSubvolumes(device, version, path string) (paths []string, err error) {
	defer func() {
		if err != nil {
			for _, p := range paths {
				syscall.Unmount(p, 0)
			}
		}
	}()
	if err := syscall.Mount(device, path, "btrfs", 0, fmt.Sprintf("subvol=%s", version)); err != nil {
		return nil, err
	}
	for k, v := range subvolumes {
		subPath := filepath.Join(path, v)
		if err := os.MkdirAll(subPath, 0755); err != nil {
			return nil, err
		}
		if err := syscall.Mount(device, subPath, "btrfs", 0, fmt.Sprintf("subvol=%s", k)); err != nil {
			return nil, errors.Wrapf(err, "mount %s:%s", k, subPath)
		}
		paths = append(paths, subPath)
	}
	paths = append(paths, path)
	return paths, nil
}

func btrfs(args ...string) error {
	out, err := exec.Command("btrfs", args...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
