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

	"github.com/BurntSushi/toml"
	v1 "github.com/stellarproject/terraos/config/v1"
	"github.com/urfave/cli"
)

var configCommand = cli.Command{
	Name:  "config",
	Usage: "generate a sample config",
	Action: func(clix *cli.Context) error {
		uid := 0
		config := &v1.Container{
			ConfigVersion: v1.Version,
			ID:            "redis-01",
			Image:         "docker.io/library/redis:alpine",
			Resources: &v1.Resources{
				CPU:    1.5,
				Memory: 1024,
				Score:  -1,
				NoFile: 1024,
			},
			GPUs: &v1.GPUs{
				Devices: []int64{
					0,
				},
				Capabilities: []string{
					"compute",
				},
			},
			Mounts: []v1.Mount{
				{
					Type:        "bind",
					Source:      "/tmp/data",
					Destination: "/data",
					Options: []string{
						"bind",
						"rw",
					},
				},
			},
			Env: []string{
				"VAR=1",
			},
			Args: []string{
				"--save",
			},
			UID:      &uid,
			GID:      &uid,
			Services: []string{"redis.io"},
			Configs: []v1.ConfigFile{
				{
					ID:   "config",
					Path: "/redis.config",
				},
			},
			Readonly: true,
			Capabilities: []string{
				"CAP_MKNOD",
			},
			Privileged: true,
			Pty:        true,
			Networks: []*v1.Network{
				{
					Type: "macvlan",
					Name: "ob0",
					IPAM: v1.IPAM{
						Type: "dhcp",
					},
				},
				{
					Type:   "bridge",
					Name:   "br0",
					Master: "eth0",
					Bridge: "eth0",
					IPAM: v1.IPAM{
						Type:        "host-local",
						Subnet:      "10.0.0.1/24",
						SubnetRange: "xxx",
						Gateway:     "10.0.0.1",
					},
				},
			},
		}
		return toml.NewEncoder(os.Stdout).Encode(config)
	},
}
