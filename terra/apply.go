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
	"errors"
	"os"

	"github.com/urfave/cli"
)

var applyCommand = cli.Command{
	Name:   "apply",
	Usage:  "apply the configuration layer to the os",
	Before: before,
	After:  after,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "http",
			Usage: "download via http",
		},
	},
	Action: func(clix *cli.Context) error {
		repo := clix.Args().First()
		if repo == "" {
			return errors.New("no configuration repo specified")
		}
		store, err := newContentStore()
		if err != nil {
			return err
		}
		defer os.RemoveAll(disk("/tmp/content"))
		// download boot images for initrd and kernel
		return applyImage(clix, store, repo, disk("config"))
	},
}
