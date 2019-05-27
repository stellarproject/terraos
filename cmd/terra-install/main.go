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

	"github.com/containerd/containerd/content"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/v1/types"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/stage0"
	"github.com/stellarproject/terraos/stage1"
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
Install terra onto a physical disk`
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
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

		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		var (
			root = "/sd"
			ctx  = cmd.CancelContext()
		)
		if err := os.MkdirAll(root, 0755); err != nil {
			return errors.Wrap(err, "create /sd")
		}

		var (
			stage0Goup *v1.DiskGroup
			groups     []*stage1.Group
			entries    []*fstab.Entry
			store      content.Store
		)
		for _, group := range node.DiskGroups {
			switch group.Stage {
			case v1.Stage0:
				stage0Goup = group
			case v1.Stage1:
				ng, err := stage1.NewGroup(group)
				if err != nil {
					return errors.Wrap(err, "new stage1 disk group")
				}
				defer ng.Close()

				groups = append(groups, ng)
			}
			if err := stage0.Format(group); err != nil {
				return errors.Wrapf(err, "format group %s", group.Label)
			}
		}
		for _, group := range groups {
			path, err := group.Init(root)
			if err != nil {
				return
			}
			// path is only different if the os volume is created
			if path != root {
				// create the content store on the osgroup
				storePath := filepath.Join(path, "terra-content")
				if store, err = image.NewContentStore(storePath); err != nil {
					return errors.Wrap(err, "new content store")
				}
				defer os.RemoveAll(storePath)
				root = path
			}
		}
		if stage0Goup != nil {
			if err := stage0.Overlay(root, stage0Goup); err != nil {
				return errors.Wrap(err, "overlay stage0 group")
			}
			closer, err := stage0.Overlay(stage0Goup)
			if err != nil {
				return err
			}
			defer closer()
		}
		if store == nil {
			return errors.New("store not created on any group")
		}
		if stage0Goup != nil {
			if err := stage0.MBR(stage0Goup.Disks[0].Device); err != nil {
				return errors.Wrap(err, "unable to install mbr")
			}
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
