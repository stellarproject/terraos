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
	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/urfave/cli"
)

var unpackCommand = cli.Command{
	Name:      "unpack",
	Usage:     "unpack an image into a location",
	ArgsUsage: "[image] [dest]",
	Action: func(clix *cli.Context) error {
		var (
			name = clix.Args().First()
			dest = clix.Args().Get(1)
			ctx  = cmd.CancelContext()
		)
		if name == "" {
			return errors.New("no image specified")
		}
		if dest == "" {
			return errors.New("no destination specified")
		}

		store, closer, err := tmpContentStore()
		if err != nil {
			return errors.Wrap(err, "unable to create content store")
		}
		defer closer()
		desc, err := image.Fetch(ctx, clix.GlobalBool("http"), store, name)
		if err != nil {
			return errors.Wrap(err, "fetch image")
		}
		if err := image.Unpack(ctx, store, desc, dest); err != nil {
			return errors.Wrap(err, "unpack image")
		}
		return nil
	},
}
