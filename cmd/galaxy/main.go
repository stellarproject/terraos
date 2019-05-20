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

	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/pkg/galaxy"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "galaxy"
	app.Version = version.Version
	app.Description = "stellar metadata server"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "enable debug output",
		},
		cli.StringFlag{
			Name:  "addr, a",
			Usage: "server address",
			Value: ":5640",
		},
		cli.StringFlag{
			Name:  "store-uri, s",
			Usage: "backend store uri",
			Value: "file:///tmp/stellar-store",
		},
	}
	app.Action = func(c *cli.Context) error {
		cfg := &galaxy.Config{
			Debug:    c.Bool("debug"),
			Addr:     c.String("addr"),
			StoreURI: c.String("store-uri"),
		}

		srv, err := galaxy.NewServer(cfg)
		if err != nil {
			return err
		}
		return srv.Run()
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
