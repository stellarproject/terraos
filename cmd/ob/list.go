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
	"os"
	"text/tabwriter"
	"time"

	units "github.com/docker/go-units"
	v1 "github.com/stellarproject/terraos/api/v1/orbit"
	"github.com/urfave/cli"
)

var listCommand = cli.Command{
	Name:  "list",
	Usage: "list containers",
	Action: func(clix *cli.Context) error {
		ctx := Context()
		agent, err := Agent(clix)
		if err != nil {
			return err
		}
		defer agent.Close()
		resp, err := agent.List(ctx, &v1.ListRequest{})
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		const tfmt = "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\n"
		fmt.Fprint(w, "ID\tIMAGE\tSTATUS\tIP\tCPU\tMEMORY\tPIDS\tSIZE\tREVISIONS\n")
		for _, c := range resp.Containers {
			fmt.Fprintf(w, tfmt,
				c.ID,
				c.Image,
				c.Status,
				c.IP,
				time.Duration(int64(c.Cpu)),
				fmt.Sprintf("%s/%s", units.HumanSize(c.MemoryUsage), units.HumanSize(c.MemoryLimit)),
				fmt.Sprintf("%d/%d", c.PidUsage, c.PidLimit),
				units.HumanSize(float64(c.FsSize)),
				len(c.Snapshots),
			)
		}
		return w.Flush()
	},
}
