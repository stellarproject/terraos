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
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"

	sigar "github.com/cloudfoundry/gosigar"
	"github.com/docker/go-units"
	"github.com/gogo/protobuf/types"
	v1 "github.com/stellarproject/terraos/api/terra/v1"
	tv1 "github.com/stellarproject/terraos/api/types/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/util"
	"github.com/urfave/cli"
)

var machineCommand = cli.Command{
	Name:  "machine",
	Usage: "manage machines",
	Subcommands: []cli.Command{
		machineRegisterCommand,
	},
	Action: func(clix *cli.Context) error {
		ctx := cmd.CancelContext()
		client, err := util.Terra(clix.GlobalString("address"))
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err := client.Machines(ctx, &types.Empty{})
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		const tfmt = "%s\t%d\t%s\n"
		fmt.Fprint(w, "UUID\tCPUS\tMEMORY\n")
		for _, m := range resp.Machines {
			fmt.Fprintf(w, tfmt,
				m.UUID,
				m.Cpus,
				units.HumanSize(float64(m.Memory*1024*1024)),
			)
		}
		return w.Flush()
	},
}

var machineRegisterCommand = cli.Command{
	Name:  "register",
	Usage: "register the current machine",
	Action: func(clix *cli.Context) error {
		var (
			ctx = cmd.CancelContext()
			cpu = sigar.CpuList{}
			mem = sigar.Mem{}
		)
		client, err := util.Terra(clix.GlobalString("address"))
		if err != nil {
			return err
		}
		defer client.Close()

		if err := cpu.Get(); err != nil {
			return err
		}
		if err := mem.Get(); err != nil {
			return err
		}
		m := &v1.RegisterRequest{
			Cpus:   uint32(len(cpu.List)),
			Memory: mem.Total / 1024 / 1024,
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
			if i.Name == "docker0" || strings.Contains(i.Name, "virbr0") {
				continue
			}
			m.NetworkDevices = append(m.NetworkDevices, &tv1.Netdev{
				Name: i.Name,
				Mac:  i.HardwareAddr.String(),
			})
		}
		resp, err := client.Register(ctx, m)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", " ")
		return enc.Encode(resp.Machine)
	},
}
