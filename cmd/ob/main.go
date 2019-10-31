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
	"context"
	"fmt"
	"os"

	"github.com/containerd/containerd/namespaces"
	raven "github.com/getsentry/raven-go"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/config"
	"github.com/stellarproject/terraos/util"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "ob"
	app.Version = version.Version
	app.Usage = "taking containers to space"
	app.Description = cmd.Banner
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.StringFlag{
			Name:   "address,a",
			Usage:  "agent address",
			Value:  "0.0.0.0",
			EnvVar: "AGENT_ADDR",
		},
		cli.IntFlag{
			Name:  "port,p",
			Usage: "agent port",
			Value: 9100,
		},
		cli.StringFlag{
			Name:   "sentry-dsn",
			Usage:  "sentry DSN",
			EnvVar: "SENTRY_DSN",
		},
	}
	app.Before = func(clix *cli.Context) error {
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if dsn := clix.GlobalString("sentry-dsn"); dsn != "" {
			raven.SetDSN(dsn)
			raven.DefaultClient.SetRelease(version.Version)
		}
		return nil
	}
	app.Commands = []cli.Command{
		checkpointCommand,
		createCommand,
		configCommand,
		deleteCommand,
		execCommand,
		getCommand,
		killCommand,
		listCommand,
		logsCommand,
		migrateCommand,
		pushCommand,
		restoreCommand,
		rollbackCommand,
		startCommand,
		stopCommand,
		updateCommand,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		raven.CaptureErrorAndWait(err, nil)
		os.Exit(1)
	}
}

func Context() context.Context {
	return namespaces.WithNamespace(context.Background(), config.DefaultNamespace)
}

func Agent(clix *cli.Context) (*util.LocalAgent, error) {
	return util.Agent(fmt.Sprintf("%s:%d", clix.GlobalString("address"), clix.GlobalInt("port")))
}
