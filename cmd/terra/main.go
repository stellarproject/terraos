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

	"github.com/getsentry/raven-go"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "terra"
	app.Version = version.Version
	app.Usage = "Terra OS Management"
	app.Description = `
                                                     ___
                                                  ,o88888
                                               ,o8888888'
                         ,:o:o:oooo.        ,8O88Pd8888"
                     ,.::.::o:ooooOoOoO. ,oO8O8Pd888'"
                   ,.:.::o:ooOoOoOO8O8OOo.8OOPd8O8O"
                  , ..:.::o:ooOoOOOO8OOOOo.FdO8O8"
                 , ..:.::o:ooOoOO8O888O8O,COCOO"
                , . ..:.::o:ooOoOOOO8OOOOCOCO"
                 . ..:.::o:ooOoOoOO8O8OCCCC"o
                    . ..:.::o:ooooOoCoCCC"o:o
                    . ..:.::o:o:,cooooCo"oo:o:
                 ` + "`" + `   . . ..:.:cocoooo"'o:o:::'
                 .` + "`" + `   . ..::ccccoc"'o:o:o:::'
                :.:.    ,c:cccc"':.:.:.:.:.'
              ..:.:"'` + "`" + `::::c:"'..:.:.:.:.:.'
            ...:.'.:.::::"'    . . . . .'
           .. . ....:."' ` + "`" + `   .  . . ''
         . . . ...."'
         .. . ."'
        .
Terra OS management`

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.StringFlag{
			Name:   "controller",
			Usage:  "controller address",
			Value:  "127.0.0.1",
			EnvVar: "TERRA_CONTROLLER",
		},
	}
	app.Before = func(clix *cli.Context) error {
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Commands = []cli.Command{
		createCommand,
		configureCommand,
		controllerCommand,
		deleteCommand,
		initCommand,
		infoCommand,
		iscsiCommand,
		provisionCommand,
		listCommand,
		pxeCommand,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var config *cmd.Terra

func Before(clix *cli.Context) error {
	t, err := cmd.LoadTerra()
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	config = t
	if t.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if t.SentryDSN != "" {
		raven.SetDSN(t.SentryDSN)
		raven.DefaultClient.SetRelease(version.Version)
	}
	return nil
}
