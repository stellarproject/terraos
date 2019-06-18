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

	"github.com/pkg/errors"
	api "github.com/stellarproject/terraos/api/controller/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

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
			return cmd.DumpNodeConfig()
		}
		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return err
		}
		address := clix.GlobalString("controller") + ":9000"
		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			return errors.Wrap(err, "dial controller")
		}
		defer conn.Close()
		client := api.NewControllerClient(conn)
		ctx := cmd.CancelContext()

		resp, err := client.Register(ctx, &api.RegisterNodeRequest{
			Node: node,
		})
		if err != nil {
			return errors.Wrap(err, "provision node from controller")
		}
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", " ")
		return e.Encode(resp)
	},
}
