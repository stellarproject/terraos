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
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/controller"
	"github.com/urfave/cli"
)

const defaultRuntime = "io.containerd.runc.v2"

var controllerCommand = cli.Command{
	Name:        "controller",
	Description: "terra infrastructure controller",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "redis",
			Usage: "redis address",
			Value: "127.0.0.1:6379",
		},
		cli.StringFlag{
			Name:  "iscsi",
			Usage: "iscsi ip",
		},
		cli.StringFlag{
			Name:  "gateway",
			Usage: "gateway address",
		},
	},
	Action: func(clix *cli.Context) error {
		ip := net.ParseIP(clix.GlobalString("controller"))
		if ip == nil || ip.To4() == nil {
			return errors.New("invalid controller ip address")
		}
		gateway := net.ParseIP(clix.String("gateway"))
		if gateway == nil || gateway.To4() == nil {
			return errors.New("invalid gateway ip address")
		}
		iscsi := net.ParseIP(clix.String("iscsi"))
		if iscsi == nil || iscsi.To4() == nil {
			return errors.New("invalid iscsi ip address")
		}
		ips := map[controller.IPType]net.IP{
			controller.Management: ip,
			controller.Gateway:    gateway,
		}
		pool := redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", clix.String("redis"))
		}, 5)
		client, err := containerd.New(
			defaults.DefaultAddress,
			containerd.WithDefaultNamespace("controller"),
			containerd.WithDefaultRuntime(defaultRuntime),
		)
		if err != nil {
			return errors.Wrap(err, "create containerd client")
		}
		controller, err := controller.New(client, ips, pool)
		if err != nil {
			return errors.Wrap(err, "new controller")
		}
		defer controller.Close()

		server := cmd.NewServer()
		v1.RegisterInfrastructureServer(server, controller)

		signals := make(chan os.Signal, 32)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-signals
			server.Stop()
		}()
		l, err := net.Listen("tcp", clix.GlobalString("controller"))
		if err != nil {
			return errors.Wrap(err, "listen tcp")
		}
		defer l.Close()

		return server.Serve(l)
	},
}
