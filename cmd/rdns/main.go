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
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "rdns"
	app.Version = version.Version
	app.Usage = "redis backed dns server"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.StringFlag{
			Name:   "sentry-dsn,s",
			Usage:  "sentry DSN",
			EnvVar: "SENTRY_DSN",
		},
		cli.StringFlag{
			Name:  "address,a",
			Usage: "dns address",
			Value: "0.0.0.0",
		},
		cli.IntFlag{
			Name:  "port,p",
			Usage: "dns port",
			Value: 53,
		},
		cli.StringFlag{
			Name:  "domain",
			Usage: "domain to serve",
			Value: "stellar",
		},
		cli.IntFlag{
			Name:  "ttl",
			Usage: "ttl for response",
			Value: 0,
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "timeout for response",
			Value: 2 * time.Second,
		},
		cli.StringSliceFlag{
			Name:  "nameserver,n",
			Usage: "recursive nameservers(include port)",
			Value: &cli.StringSlice{
				"8.8.8.8:53",
				"8.8.4.4:53",
			},
		},
		cli.StringFlag{
			Name:  "redis,r",
			Usage: "redis address",
			Value: "127.0.0.1:9300",
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
	app.Action = func(clix *cli.Context) error {
		pool := redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", clix.GlobalString("redis"))
		}, 5)

		srv := New(pool)

		srv.IP = clix.GlobalString("address")
		srv.Port = clix.GlobalInt("port")
		srv.TTL = uint32(clix.GlobalInt("ttl"))
		srv.Timeout = clix.GlobalDuration("timeout")
		srv.UDPSize = 65535
		srv.Nameservers = clix.GlobalStringSlice("nameserver")
		srv.Domain = clix.GlobalString("domain")

		ctx := cancelContext()
		return srv.Serve(ctx)
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		raven.CaptureErrorAndWait(err, nil)
		os.Exit(1)
	}
}

func cancelContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	s := make(chan os.Signal)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-s
		cancel()
	}()
	return ctx
}
