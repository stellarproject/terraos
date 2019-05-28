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
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/namespaces"
	raven "github.com/getsentry/raven-go"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/agent"
	v1 "github.com/stellarproject/terraos/api/v1/services"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/config"
	"github.com/stellarproject/terraos/util"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	defaultRuntime = "io.containerd.runc.v2"
	configPath     = "/etc/stellar/orbit.toml"
)

func main() {
	app := cli.NewApp()
	app.Name = "orbit-server"
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
			Value:  "0.0.0.0:9100",
			EnvVar: "AGENT_ADDR",
		},
		cli.StringFlag{
			Name:   "sentry-dsn",
			Usage:  "sentry DSN",
			EnvVar: "SENTRY_DSN",
		},
		cli.DurationFlag{
			Name:  "interval,i",
			Usage: "agent interval loop",
			Value: 10 * time.Second,
		},
		cli.StringFlag{
			Name:  "id",
			Usage: "id of the agent/node",
		},
		cli.StringFlag{
			Name:  "domain",
			Usage: "domain of the agent/node",
		},
		cli.StringFlag{
			Name:  "iface",
			Usage: "external interface of the agent/node",
		},
		cli.StringFlag{
			Name:  "state",
			Usage: "state directory",
			Value: "/run/orbit",
		},
		cli.StringSliceFlag{
			Name:  "plain-remote",
			Usage: "http registries",
			Value: &cli.StringSlice{},
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
		c := &agent.Config{
			ID:           clix.GlobalString("id"),
			Iface:        clix.GlobalString("iface"),
			State:        clix.GlobalString("state"),
			ClusterDir:   "/cluster",
			Interval:     clix.GlobalDuration("interval"),
			PlainRemotes: clix.GlobalStringSlice("plain-remote"),
		}
		if c.Iface == "" {
			i, err := util.GetDefaultIface()
			if err != nil {
				if err != util.ErrNoDefaultRoute {
					return errors.Wrap(err, "getting default iface")
				}
				i = "lo"
			}
			c.Iface = i
		}
		if c.Domain == "" {
			d, err := util.GetDomainName()
			if err != nil {
				return errors.Wrap(err, "get domain name")
			}
			c.Domain = d
		}
		if c.ID == "" {
			h, err := os.Hostname()
			if err != nil {
				return errors.Wrap(err, "get hostname")
			}
			c.ID = h
		}
		if err := os.MkdirAll(c.State, 0711); err != nil {
			return errors.Wrap(err, "create state directory")
		}

		client, err := containerd.New(
			defaults.DefaultAddress,
			containerd.WithDefaultNamespace(config.DefaultNamespace),
			containerd.WithDefaultRuntime(defaultRuntime),
		)
		if err != nil {
			return errors.Wrap(err, "create containerd client")
		}
		ctx, cancel := context.WithCancel(namespaces.WithNamespace(context.Background(), config.DefaultNamespace))

		a, err := agent.New(ctx, c, client)
		if err != nil {
			return errors.Wrap(err, "new agent")
		}

		server := newServer()
		v1.RegisterAgentServer(server, a)

		signals := make(chan os.Signal, 32)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-signals
			cancel()
			server.Stop()
		}()
		l, err := net.Listen("tcp", clix.GlobalString("address"))
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

func newServer() *grpc.Server {
	s := grpc.NewServer(
		grpc.UnaryInterceptor(unary),
		grpc.StreamInterceptor(stream),
	)
	hs := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, hs)
	return s
}

func unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	r, err := grpc_prometheus.UnaryServerInterceptor(ctx, req, info, handler)
	if err != nil {
		raven.CaptureError(err, nil)
	}
	return r, err
}

func stream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := grpc_prometheus.StreamServerInterceptor(srv, ss, info, handler)
	if err != nil {
		raven.CaptureError(err, nil)
	}
	return err
}
