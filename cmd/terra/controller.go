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

	"github.com/BurntSushi/toml"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/v1/infra"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/controller"
	"github.com/stellarproject/terraos/util"
	"github.com/urfave/cli"
)

const (
	defaultRuntime = "io.containerd.runc.v2"
	configPath     = "/etc/terra/controller.toml"
)

type Config struct {
	Redis      string `toml:"redis"`
	Controller string `toml:"controller"`
	ISCSI      string `toml:"iscsi"`
	Gateway    string `toml:"gateway"`
}

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
		cli.BoolFlag{
			Name:  "etcd",
			Usage: "enabled the managed etc",
		},
		// deprecated
		cli.StringSliceFlag{
			Name:   "plain",
			Usage:  "specify plain registry remotes",
			Value:  &cli.StringSlice{},
			Hidden: true,
		},
	},
	Action: func(clix *cli.Context) error {
		logrus.Info("loading config...")
		c, err := loadControllerConfig(clix)
		if err != nil {
			return err
		}
		logrus.Info("building ip map...")
		ip := net.ParseIP(c.Controller)
		if ip == nil || ip.To4() == nil {
			return errors.New("invalid controller ip address")
		}
		gateway := net.ParseIP(c.Gateway)
		if gateway == nil || gateway.To4() == nil {
			return errors.New("invalid gateway ip address")
		}
		iscsi := net.ParseIP(c.ISCSI)
		if iscsi == nil || iscsi.To4() == nil {
			return errors.New("invalid iscsi ip address")
		}
		ips := map[controller.IPType]net.IP{
			controller.Management: ip,
			controller.Gateway:    gateway,
			controller.ISCSI:      iscsi,
		}
		logrus.Info("creating redis pool...")
		pool := redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", c.Redis)
		}, 5)
		logrus.Info("connecting to orbit...")
		orbit, err := util.Agent("127.0.0.1:9100")
		if err != nil {
			return errors.Wrap(err, "get orbit agent")
		}
		logrus.Info("connecting to containerd...")
		client, err := containerd.New(
			defaults.DefaultAddress,
			containerd.WithDefaultNamespace("controller"),
			containerd.WithDefaultRuntime(defaultRuntime),
		)
		if err != nil {
			return errors.Wrap(err, "create containerd client")
		}
		logrus.Info("creating new controller...")
		controller, err := controller.New(client, controller.Config{
			IPConfig: ips,
			Pool:     pool,
			Orbit:    orbit,
		})
		if err != nil {
			return errors.Wrap(err, "new controller")
		}
		defer controller.Close()

		logrus.Info("registering grpc server...")
		server := cmd.NewServer()
		v1.RegisterControllerServer(server, controller)

		signals := make(chan os.Signal, 32)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-signals
			server.Stop()
		}()

		logrus.Info("listening on controller address...")
		l, err := net.Listen("tcp", c.Controller+":9000")
		if err != nil {
			return errors.Wrap(err, "listen tcp")
		}
		defer l.Close()

		logrus.Info("serving controller api...")
		logrus.Info("controller boot successful...")
		defer logrus.Info("controller exiting, bye bye...")

		return server.Serve(l)
	},
}

func loadControllerConfig(clix *cli.Context) (*Config, error) {
	c := Config{
		Redis:      clix.String("redis"),
		Controller: clix.GlobalString("controller"),
		ISCSI:      clix.String("iscsi"),
		Gateway:    clix.String("gateway"),
	}
	if _, err := toml.DecodeFile(configPath, &c); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "load config")
		}
	}
	return &c, nil
}
