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
	"github.com/urfave/cli"
)

var configCommand = cli.Command{
	Name:  "config",
	Usage: "manage configs",
	Subcommands: []cli.Command{
		configAddCommand,
		configGetCommand,
	},
	Action: func(clix *cli.Context) error {
		return nil
		/*
			store := getCluster(clix)
			ctx := cmd.CancelContext()
			configs, err := store.Configs().List(ctx)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
			const tfmt = "%s\t%s\n"
			fmt.Fprint(w, "ID\tPATH\n")
			for _, c := range configs {
				fmt.Fprintf(w, tfmt,
					c.ID,
					c.Path,
				)
			}
			return w.Flush()
		*/
	},
}

var configGetCommand = cli.Command{
	Name:  "get",
	Usage: "get a config",
	Action: func(clix *cli.Context) error {
		return nil
		/*
			id := clix.Args().First()
			if id == "" {
				return errors.New("no config id")
			}
			store := getCluster(clix)
			ctx := cmd.CancelContext()
			c, err := store.Configs().Get(ctx, id)
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(c.Contents)
			return err
		*/
	},
}
var configAddCommand = cli.Command{
	Name:  "add",
	Usage: "add a config",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "stdin,i",
			Usage: "get data from stdin",
		},
	},
	Action: func(clix *cli.Context) error {
		return nil
		/*
			id := clix.Args().First()
			if id == "" {
				return errors.New("no config id")
			}
			path := clix.Args().Get(1)
			if path == "" {
				return errors.New("no config path")
			}
			var (
				data []byte
				err  error
			)
			rd := clix.Args().Get(2)
			if rd != "" {
				data = []byte(rd)
			}
			if clix.Bool("stdin") {
				if data, err = ioutil.ReadAll(os.Stdin); err != nil {
					return errors.Wrap(err, "reading stdin")
				}
			}
			store := getCluster(clix)
			ctx := cmd.CancelContext()
			c := &v1.Config{
				ID:       id,
				Path:     path,
				Contents: data,
			}
			return store.Configs().Save(ctx, c)
		*/
	},
}
