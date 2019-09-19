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
	"os"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/resolvconf"
	"github.com/urfave/cli"
)

var defaultNamservers = []string{
	"1.1.1.1",
	"1.0.0.1",
}

var clusterInitDirs = []string{
	"nodes",
	"networks",
}

const clusterFilename = "TERRA_CLUSTER"

var clusterCommand = cli.Command{
	Name:  "cluster",
	Usage: "perform cluster operations",
	Subcommands: []cli.Command{
		nodeCommand,
		initCommand,
		networkCommand,
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "init a cluster repo",
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "nameserver,n",
			Usage: "specify the default nameservers",
			Value: &cli.StringSlice{},
		},
	},
	Action: func(clix *cli.Context) error {
		name := clix.Args().First()
		if name == "" {
			return errors.New("cluster name must be specified")
		}
		nameservers := clix.StringSlice("nameserver")
		if len(nameservers) == 0 {
			nameservers = defaultNamservers
		}
		ctx := cmd.CancelContext()
		if err := os.Mkdir(name, 0755); err != nil {
			return errors.Wrap(err, "create cluster dir")
		}
		if err := os.Chdir(name); err != nil {
			return errors.Wrap(err, "chdir into cluster")
		}
		if err := git(ctx, "init"); err != nil {
			return errors.Wrap(err, "init git repository")
		}
		if err := createTerraClusterFile(); err != nil {
			return errors.Wrap(err, "create terra cluster file")
		}
		if err := writeNameservers(ctx, nameservers); err != nil {
			return errors.Wrap(err, "write nameservers")
		}
		if err := createDirs(clusterInitDirs, true); err != nil {
			return err
		}
		if err := git(ctx, "add", "-A"); err != nil {
			return errors.Wrap(err, "add all files to git")
		}
		if err := commit(ctx, "init cluster"); err != nil {
			return errors.Wrap(err, "commit repo")
		}
		return nil
	},
}

func writeNameservers(ctx context.Context, ns []string) error {
	resolv := &resolvconf.Conf{
		Nameservers: ns,
	}
	f, err := os.Create("resolv.conf")
	if err != nil {
		return err
	}
	defer f.Close()
	return resolv.Write(f)
}
