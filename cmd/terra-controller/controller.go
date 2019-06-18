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
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/getsentry/raven-go"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/controller/v1"
	pxev1 "github.com/stellarproject/terraos/api/pxe/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/services/controller"
	"github.com/stellarproject/terraos/services/pxe"
	"github.com/urfave/cli"
	"honnef.co/go/tools/version"
)

func main() {
	app := cli.NewApp()
	app.Name = "terra-controller"
	app.Version = version.Version
	app.Usage = "Terra Controller Server"
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
`
	var config *cmd.Terra
	app.Before = func(clix *cli.Context) error {
		t, err := cmd.LoadTerra()
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		if t.Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if t.SentryDSN != "" {
			raven.SetDSN(t.SentryDSN)
			raven.DefaultClient.SetRelease(version.Version)
		}
		return nil
	}
	app.Action = func(clix *cli.Context) error {
		pool := redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", config.Redis)
		}, 5)
		defer pool.Close()

		logrus.Info("creating new controller...")
		controller, err := controller.New(pool, config.SSHKeys)
		if err != nil {
			return errors.Wrap(err, "new controller")
		}
		store, err := image.NewContentStore("/content")
		if err != nil {
			return errors.Wrap(err, "create content store")
		}
		pxeServer, err := pxe.New(pool, store)
		if err != nil {
			return errors.Wrap(err, "create pxe server")
		}

		server := cmd.NewServer()
		v1.RegisterControllerServer(server, controller)
		pxev1.RegisterServiceServer(server, pxeServer)

		signals := make(chan os.Signal, 32)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-signals
			server.Stop()
		}()

		logrus.Info("listening on controller address...")
		l, err := net.Listen("tcp", config.Addr())
		if err != nil {
			return errors.Wrap(err, "listen tcp")
		}
		defer l.Close()

		return server.Serve(l)
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		raven.CaptureErrorAndWait(err, nil)
		os.Exit(1)
	}
}
