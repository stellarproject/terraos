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

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var directories = []string{
	"boot",
	"tmp",
}

var installCommand = cli.Command{
	Name:   "install",
	Usage:  "install terra on your system",
	Before: before,
	After:  after,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "http",
			Usage: "download via http",
		},
		cli.StringFlag{
			Name:  "config",
			Usage: "config image to install",
		},
	},
	Action: func(clix *cli.Context) error {
		version, err := getVersion(clix)
		if err != nil {
			return err
		}
		logger := logrus.WithField("version", version)
		logger.Info("setup directories")
		if err := setupDirectories(); err != nil {
			return err
		}

		logger.Info("creating new content store")
		store, err := newContentStore(disk())
		if err != nil {
			return err
		}

		ctx := cancelContext()
		osDesc, err := fetch(ctx, clix.Bool("http"), store, fmt.Sprintf(terraRepoFormat, version))
		if err != nil {
			return err
		}
		// unpack the os as a
		sn, err := newSnapshotter(disk())
		if err != nil {
			return err
		}
		defer sn.Close()

		chain, err := unpackSnapshots(ctx, store, sn, osDesc)
		if err != nil {
			return err
		}
		// create config layer
		if _, err := sn.Prepare(ctx, configRWKey, chain); err != nil {
			return err
		}
		configImage := clix.String("config")
		if configImage != "" {
			if err := applyConfig(ctx, clix.Bool("http"), store, sn, configImage); err != nil {
				return err
			}
		}
		if err := sn.Commit(ctx, configKey, configRWKey); err != nil {
			return err
		}
		mounts, err := sn.Prepare(ctx, version, configKey)
		if err != nil {
			return err
		}
		return writeMountOptions(mounts[0].Options)
	},
}

func setupDirectories() error {
	for _, d := range directories {
		if err := os.MkdirAll(disk(d), 0755); err != nil {
			return err
		}
	}
	return nil
}
