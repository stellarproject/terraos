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
	api "github.com/stellarproject/terraos/api/v1/services"
	v1 "github.com/stellarproject/terraos/config/v1"
	"github.com/urfave/cli"
)

var updateCommand = cli.Command{
	Name:  "update",
	Usage: "update an existing container's configuration",
	Action: func(clix *cli.Context) error {
		var (
			path = clix.Args().First()
			ctx  = Context()
		)
		var newConfig v1.Container
		if _, err := toml.DecodeFile(path, &newConfig); err != nil {
			return err
		}
		agent, err := Agent(clix)
		if err != nil {
			return err
		}
		defer agent.Close()
		c, err := newConfig.Proto()
		if err != nil {
			return err
		}
		_, err = agent.Update(ctx, &api.UpdateRequest{
			Container: c,
		})
		return err
	},
}
