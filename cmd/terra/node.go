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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var nodeDirectories = []string{
	"fs/etc",
	"interfaces",
}

var nodeCommand = cli.Command{
	Name:  "node",
	Usage: "manage nodes",
	Subcommands: []cli.Command{
		nodeInitCommand,
		nodeAttachNetworkCommand,
	},
	Action: func(clix *cli.Context) error {
		dirs, err := ioutil.ReadDir("nodes")
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		const tfmt = "%s\n"
		fmt.Fprint(w, "HOSTNAME\n")
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			fmt.Fprintf(w, tfmt, d.Name())
		}
		return w.Flush()
	},
}

var nodeAttachNetworkCommand = cli.Command{
	Name:  "network-attach",
	Usage: "attach a node to a network",
	Action: func(clix *cli.Context) error {
		name := clix.Args().First()
		if name == "" {
			return errors.New("name must be specified")
		}
		name = strings.ToUpper(name)
		iface := clix.Args().Get(1)
		if iface == "" {
			dirs, err := ioutil.ReadDir("interfaces")
			if err != nil {
				return err
			}
			for _, d := range dirs {
				if d.IsDir() {
					iface = d.Name()
				}
			}
		}
		if iface == "" {
			return errors.New("no interface name specified")
		}
		address, err := ioutil.ReadFile(filepath.Join("../../networks", name, "address"))
		if err != nil {
			return err
		}
		i, ip := allocateIP(filepath.Join("../../networks", name), string(address))
		if i == -1 {
			return errors.New("could not allocate ip")
		}
		if err := os.Symlink(filepath.Join("../../networks", name, ip), filepath.Join("interfaces", iface, ip)); err != nil {
			return err
		}
		return nil
	},
}

var nodeInitCommand = cli.Command{
	Name:  "init",
	Usage: "init a node",
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "interface,i",
			Usage: "name of network interface",
			Value: &cli.StringSlice{},
		},
	},
	Action: func(clix *cli.Context) error {
		name := clix.Args().First()
		if name == "" {
			return errors.New("node name must be specified")
		}
		path := filepath.Join("nodes", name)
		if err := os.Mkdir(path, 0755); err != nil {
			return errors.New("create node dir")
		}
		if err := os.Chdir(path); err != nil {
			return errors.Wrap(err, "chdir into node")
		}
		if err := createDirs(nodeDirectories, true); err != nil {
			return err
		}
		if err := os.Symlink("../../../../resolv.conf", "fs/etc/resolv.conf"); err != nil {
			return errors.Wrap(err, "resolv.conf symlink")
		}
		for _, i := range clix.StringSlice("interface") {
			if err := os.Mkdir(filepath.Join("interfaces", i), 0755); err != nil {
				return err
			}
		}
		return nil
	},
}
