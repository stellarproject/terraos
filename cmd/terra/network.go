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
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "manage networks",
	Subcommands: []cli.Command{
		networkInitCommand,
	},
}

type networkType string

const (
	vlan networkType = "vlan"
	lan  networkType = "lan"
	wan  networkType = "wan"
)

var networkDirs = []string{}

var networkInitCommand = cli.Command{
	Name:  "init",
	Usage: "init a network",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "type",
			Usage: "network type",
			Value: string(lan),
		},
	},
	Action: func(clix *cli.Context) error {
		name := clix.Args().First()
		if name == "" {
			return errors.New("network name must be specified")
		}
		name = strings.ToUpper(name)
		address := clix.Args().Get(1)
		if address == "" {
			return errors.New("address must be specified")
		}
		path := filepath.Join("networks", name)
		if err := os.Mkdir(path, 0755); err != nil {
			return errors.New("create network dir")
		}
		if err := os.Chdir(path); err != nil {
			return errors.Wrap(err, "chdir into network")
		}
		if err := writeFile("address", address); err != nil {
			return err
		}
		if err := writeFile(strings.ToUpper(clix.String("type")), ""); err != nil {
			return err
		}
		return nil
	},
}
