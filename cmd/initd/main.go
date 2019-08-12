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
	"os/signal"
	"runtime"

	"github.com/containerd/containerd/sys"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
	"golang.org/x/sys/unix"
)

func main() {
	runtime.LockOSThread()

	app := cli.NewApp()
	app.Name = "initd"
	app.Version = version.Version
	app.Usage = "Linux Init Daemon"
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
Terra OS Init Daemon for Linux`

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
	}
	app.Before = func(clix *cli.Context) error {
		logrus.Infof("booting initd version %s", version.Version)
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Action = func(clix *cli.Context) error {
		sigs := make(chan os.Signal, 1024)
		signal.Notify(sigs)

		pid := os.Getpid()
		if pid != 1 {
			logrus.Infof("not running as PID 1(%d), setting sub-reaper", pid)
			if err := sys.SetSubreaper(1); err != nil {
				return err
			}
		}

		// run handlers
		for s := range sigs {
			switch s {
			case unix.SIGCHLD:
				exits, err := sys.Reap(false)
				if err != nil {
					logrus.WithError(err).Error("reaping children")
				}
				for _, e := range exits {
					logrus.WithFields(logrus.Fields{
						"pid":    e.Pid,
						"status": e.Status,
					}).Debug("process exit")
				}
			case unix.SIGTERM, unix.SIGINT:
				logrus.Info("shutdown signaled via (%d)", s)
				// TODO: graceful stop of all children
				return nil
			}
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
