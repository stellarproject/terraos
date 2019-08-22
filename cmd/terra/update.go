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
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/urfave/cli"
)

var updateCommand = cli.Command{
	Name:  "update",
	Usage: "update a package on the terra machine",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "http",
			Usage: "fetch image over http",
		},
		cli.StringFlag{
			Name:  "dest,d",
			Usage: "destination to install",
			Value: "/",
		},
	},
	Action: func(clix *cli.Context) error {
		img := clix.Args().First()
		if img == "" {
			return errors.New("no image specified to install")
		}
		ctx := cmd.CancelContext()

		// handle running on a provisioning machine vs in the iso
		storePath := filepath.Join("/tmp", "contentstore")
		if _, err := os.Stat(contentStorePath); err == nil {
			storePath = contentStorePath
		}

		store, err := image.NewContentStore(storePath)
		if err != nil {
			return errors.Wrap(err, "new content store")
		}

		desc, err := image.Fetch(ctx, clix.Bool("http"), store, img)
		if err != nil {
			return errors.Wrap(err, "fetch image")
		}

		dest := clix.String("dest")
		if err := os.MkdirAll(dest, 0755); err != nil {
			return errors.Wrap(err, "mkdir dest")
		}

		// unpack image onto the destination
		if err := image.Unpack(ctx, store, desc, dest); err != nil {
			return errors.Wrap(err, "unpack image")
		}
		return nil
	},
}
