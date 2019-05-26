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
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/v1/services"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/disk"
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
			Value: "btrfs",
		},
		cli.StringSliceFlag{
			Name:  "subvolumes,s",
			Usage: "persistent subvolumes format> name:path",
			Value: &cli.StringSlice{},
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
			path   = "/sd"
			ctx    = cmd.CancelContext()
			fstype = clix.GlobalString("fs-type")
		)
		if err := os.MkdirAll(path, 0755); err != nil {
			return errors.Wrap(err, "create /sd")
		}
		node := &v1.Node{
			Image: string(repo),
			Fs: &v1.Filesystem{
				Subvolumes: parseSubvolumes(clix.GlobalStringSlice("subvolumes")),
			},
		}
		d := disk.NewLocalDisk(clix.GlobalString("device"))
		if err := d.Format(ctx, fstype, "os"); err != nil {
			return errors.Wrap(err, "format disk")
		}
		if err := d.Provision(ctx, fstype, node); err != nil {
			d.Unmount(ctx)
			return errors.Wrap(err, "provision disk")
		}
		storePath := filepath.Join(path, "terra-content")
		store, err := image.NewContentStore(storePath)
		if err != nil {
			return errors.Wrap(err, "new content store")
		}
		if err := d.Write(ctx, image.Repo(node.Image), store, nil); err != nil {
			os.RemoveAll(storePath)
			d.Unmount(ctx)
			return errors.Wrap(err, "write image to disk")
		}
		if err := os.RemoveAll(storePath); err != nil {
			d.Unmount(ctx)
			return errors.Wrap(err, "remove store path")
		}
		if err := d.Unmount(ctx); err != nil {
			return errors.Wrap(err, "unmount disk")
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseSubvolumes(raw []string) (out []*v1.Subvolume) {
	for _, s := range raw {
		parts := strings.SplitN(s, ":", 2)
		out = append(out, &v1.Subvolume{
			Name: parts[0],
			Path: parts[1],
		})
	}
	return out
}
