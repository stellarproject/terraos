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
	"net"

	sigar "github.com/cloudfoundry/gosigar"
	v1 "github.com/stellarproject/terraos/api/cluster/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/urfave/cli"
)

var machineCommand = cli.Command{
	Name:  "machine",
	Usage: "manage machines",
	Subcommands: []cli.Command{
		machineRegisterCommand,
	},
	Action: func(clix *cli.Context) error {
		return nil
	},
}

var machineRegisterCommand = cli.Command{
	Name:  "register",
	Usage: "register the current machine",
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "machine labels",
			Value: &cli.StringSlice{},
		},
	},
	Action: func(clix *cli.Context) error {
		store := getCluster(clix)
		ctx := cmd.CancelContext()
		cluster, err := store.Get(ctx)
		if err != nil {
			return err
		}
		cpu := sigar.CpuList{}
		if err := cpu.Get(); err != nil {
			return err
		}
		mem := sigar.Mem{}
		if err := mem.Get(); err != nil {
			return err
		}
		m := &v1.Machine{
			Cpus:   uint32(len(cpu.List)),
			Memory: mem.Total / 1024 / 1024,
			Labels: clix.StringSlice("label"),
		}
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}
		const skipFlags = net.FlagPointToPoint | net.FlagLoopback
		for _, i := range interfaces {
			if i.Flags&skipFlags != 0 {
				continue
			}
			m.NetworkDevices = append(m.NetworkDevices, &v1.Netdev{
				Name: i.Name,
				Mac:  i.HardwareAddr.String(),
			})
		}
		if err := cluster.RegisterMachine(ctx, m); err != nil {
			return err
		}
		return store.Commit(ctx, cluster)
	},
}
