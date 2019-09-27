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
	"strings"

	v1 "github.com/stellarproject/terraos/api/cluster/v1"
	"github.com/urfave/cli"
)

var volumeCommand = cli.Command{
	Name:  "volume",
	Usage: "manage volumes",
	Subcommands: []cli.Command{
		volumeAddCommand,
	},
	Action: func(clix *cli.Context) error {
		return nil

		/*
			store := getCluster(clix)
			ctx := cmd.CancelContext()
			volumes, err := store.Volumes().List(ctx)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
			const tfmt = "%s\t%d\t%s\t%s\n"
			fmt.Fprint(w, "ID\tLUN\tPATH\tLABEL\n")
			for _, v := range volumes {
				for i, l := range v.Luns {
					fmt.Fprintf(w, tfmt,
						v.ID,
						i,
						l.Path,
						l.Label,
					)
				}
			}
			return w.Flush()
		*/
	},
}

var volumeAddCommand = cli.Command{
	Name:  "add",
	Usage: "add a volume",
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "lun",
			Usage: "lun info",
			Value: &cli.StringSlice{},
		},
	},
	Action: func(clix *cli.Context) error {
		return nil
		/*
			store := getCluster(clix)
			ctx := cmd.CancelContext()
			id := clix.Args().First()
			v := &v1.Volume{
				ID: id,
			}
			for i, s := range clix.StringSlice("lun") {
				v.Luns = append(v.Luns, parseLun(i, s))
			}
			return store.Volumes().Save(ctx, v)
		*/
	},
}

func parseLun(i int, s string) *v1.Lun {
	parts := strings.SplitN(s, ":", 2)
	return &v1.Lun{
		Path:  parts[0],
		Label: parts[1],
	}
}