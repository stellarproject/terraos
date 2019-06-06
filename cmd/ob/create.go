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
	"github.com/BurntSushi/toml"
	api "github.com/stellarproject/terraos/api/v1/orbit"
	v1 "github.com/stellarproject/terraos/config/v1"
	"github.com/urfave/cli"
)

var createCommand = cli.Command{
	Name:  "create",
	Usage: "create a container",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "update",
			Usage: "create or update",
		},
	},
	Action: func(clix *cli.Context) error {
		var container v1.Container
		if _, err := toml.DecodeFile(clix.Args().First(), &container); err != nil {
			return err
		}
		agent, err := Agent(clix)
		if err != nil {
			return err
		}
		defer agent.Close()
		c, err := container.Proto()
		if err != nil {
			return err
		}
		_, err = agent.Create(Context(), &api.CreateRequest{
			Container: c,
			Update:    clix.Bool("update"),
		})
		return err
	},
}
