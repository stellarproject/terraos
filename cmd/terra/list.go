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
	"os"
	"text/tabwriter"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1/infra"
	"github.com/stellarproject/terraos/cmd"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

var listCommand = cli.Command{
	Name:        "list",
	Description: "list all registered nodes",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "json",
			Usage: "output in json format",
		},
	},
	Action: func(clix *cli.Context) error {
		address := clix.GlobalString("controller") + ":9000"
		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			return errors.Wrap(err, "dial controller")
		}
		defer conn.Close()
		client := v1.NewControllerClient(conn)
		ctx := cmd.CancelContext()

		resp, err := client.List(ctx, &types.Empty{})
		if err != nil {
			return errors.Wrap(err, "list nodes from terra")
		}
		if clix.Bool("json") {
			return json.NewEncoder(os.Stdout).Encode(resp.Nodes)
		}
		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		const tfmt = "%s\t%s\t%s\t%s\t%s\n"
		fmt.Fprint(w, "HOSTNAME\tMAC\tIMAGE\tINITIATOR\tTARGET\n")
		for _, n := range resp.Nodes {
			var iqn string
			if n.DiskGroups[0].Target != nil {
				iqn = n.DiskGroups[0].Target.Iqn
			}
			fmt.Fprintf(w, tfmt,
				n.Hostname,
				n.Mac,
				n.Image,
				n.InitiatorIqn,
				iqn,
			)
		}
		return w.Flush()
	},
}
