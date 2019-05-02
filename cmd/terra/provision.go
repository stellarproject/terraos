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
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/urfave/cli"
)

var provisionCommand = cli.Command{
	Name:  "provision,p",
	Usage: "provision a disk for terra",
	Flags: []cli.Flag{
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
	},
	Action: func(clix *cli.Context) error {
		var (
			t      = clix.String("fs-type")
			device = clix.String("device")
		)
		switch t {
		case "ext4":
			return mkfs.Init(t, device, "terra")
		case "btrfs":
			if err := mkfs.Init(t, device, "terra"); err != nil {
				return err
			}
		default:
			return errors.New("unknown fs type")
		}
		return nil
	},
}
