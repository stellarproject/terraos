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

	"github.com/containerd/containerd/snapshots"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var deleteCommand = cli.Command{
	Name:   "delete",
	Usage:  "delete a release",
	Before: before,
	After:  after,
	Action: func(clix *cli.Context) error {
		version, err := getVersion(clix)
		if err != nil {
			return err
		}
		logger := logrus.WithField("version", version)
		ctx := cancelContext()
		sn, err := newSnapshotter(disk())
		if err != nil {
			return err
		}
		defer sn.Close()
		parents := make(map[string]string)
		if err := sn.Walk(ctx, func(ctx context.Context, info snapshots.Info) error {
			parents[info.Name] = info.Parent
			return nil
		}); err != nil {
			return err
		}
		current := version
		for {
			if current == "" {
				break
			}
			logger.Infof("removing %s...", current)
			if err := sn.Remove(ctx, current); err != nil {
				return err
			}
			current = parents[current]

		}
		return nil
	},
}
