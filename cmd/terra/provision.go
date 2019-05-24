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
	"encoding/json"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

type Subvolume struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
}

type Node struct {
	Hostname   string      `toml:"hostname"`
	MAC        string      `toml:"mac"`
	Image      string      `toml:"image"`
	BackingURI string      `toml:"fs_uri"`
	Size       int64       `toml:"fs_size"`
	Subvolumes []Subvolume `toml:"fs_subvolumes"`
}

var provisionCommand = cli.Command{
	Name:        "provision",
	Description: "provision a new node",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "dump",
			Usage: "dump and example config",
		},
	},
	Action: func(clix *cli.Context) error {
		if clix.Bool("dump") {
			return dumpNodeConfig()
		}
		var node Node
		if _, err := toml.DecodeFile(clix.Args().First(), &node); err != nil {
			return errors.Wrap(err, "load node file")
		}

		address := clix.GlobalString("controller") + ":9000"
		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			return errors.Wrap(err, "dial controller")
		}
		defer conn.Close()
		client := v1.NewInfrastructureClient(conn)
		ctx := cmd.CancelContext()

		resp, err := client.Provision(ctx, &v1.ProvisionNodeRequest{
			Hostname: node.Hostname,
			Mac:      node.MAC,
			Image:    node.Image,
			Fs: &v1.Filesystem{
				BackingUri: node.BackingURI,
				FsSize:     node.Size,
				Subvolumes: subvolumes(node.Subvolumes),
			},
		})
		if err != nil {
			return errors.Wrap(err, "provision node from controller")
		}
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", " ")
		return e.Encode(resp.Node)
	},
}

func subvolumes(subvolumes []Subvolume) (out []*v1.Subvolume) {
	for _, s := range subvolumes {
		out = append(out, &v1.Subvolume{
			Name: s.Name,
			Path: s.Path,
		})
	}
	return out
}

func dumpNodeConfig() error {
	c := &Node{
		Hostname:   "terra-01",
		MAC:        "66:xx:ss:bb:f1:b1",
		Image:      "docker.io/stellarproject/example:v4",
		BackingURI: "iscsi://btrfs",
		Size:       512,
		Subvolumes: []Subvolume{
			{
				Name: "tftp",
				Path: "/tftp",
			},
		},
	}
	return toml.NewEncoder(os.Stdout).Encode(c)
}
