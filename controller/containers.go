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

package controller

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/containerd/typeurl"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/v1/services"
	"github.com/stellarproject/terraos/opts"
	"github.com/stellarproject/terraos/util"
)

const (
	redisID    = "controller-redis"
	promID     = "controller-prometheus"
	registryID = "controller-registry"
)

type redisContainer struct {
	ip    net.IP
	orbit *util.LocalAgent
}

func (r *redisContainer) Start(ctx context.Context) error {
	if _, err := r.orbit.Get(ctx, &v1.GetRequest{
		ID: redisID,
	}); err == nil {
		logrus.Debug("existing redis container is running")
		return nil
	}

	logrus.Info("starting redis container")
	args := []string{
		"docker-entrypoint.sh",
		"redis-server",
		"--appendonly", "yes",
		"--bind", r.ip.To4().String(),
	}
	if r.ip.To4().String() != "127.0.0.1" {
		args = append(args, "127.0.0.1")
	}
	container := &v1.Container{
		ID:    redisID,
		Image: "docker.io/library/redis:5-alpine",
		Process: &v1.Process{
			Args: args,
		},
	}
	any, err := typeurl.MarshalAny(&v1.HostNetwork{})
	if err != nil {
		panic(err)
	}
	container.Networks = append(container.Networks, any)

	container.Resources = &v1.Resources{
		NoFile: 2048,
		Cpus:   1.5,
		Memory: 1024,
	}
	if _, err := r.orbit.Create(ctx, &v1.CreateRequest{
		Container: container,
	}); err != nil {
		return errors.Wrap(err, "create redis container")
	}
	return nil
}

const registryConfig = `version: 0.1
log:
  level: info
  fields:
    service: registry
    environment: prod
storage:
    delete:
      enabled: true
    cache:
        blobdescriptor: inmemory
    filesystem:
        rootdirectory: /var/lib/registry
    maintenance:
        uploadpurging:
            enabled: false
http:
    addr: %s:80
    debug:
        addr: %s:5001
        prometheus:
            enabled: true
            path: /metrics
    headers:
        X-Content-Type-Options: [nosniff]`

const registryConfigFilename = "controller-registry.yaml"

type registryContainer struct {
	ip    net.IP
	orbit *util.LocalAgent
}

func (r *registryContainer) Start(ctx context.Context) error {
	if _, err := r.orbit.Get(ctx, &v1.GetRequest{
		ID: registryID,
	}); err == nil {
		logrus.Debug("existing registry container is running")
		return nil
	}

	paths := opts.Paths{
		Cluster: ClusterFS,
	}
	f, err := os.Create(paths.ConfigPath(registryConfigFilename))
	if err != nil {
		return errors.Wrap(err, "create registry config")
	}
	ip := r.ip.To4().String()
	_, err = fmt.Fprintf(f, registryConfig, ip, ip)
	f.Close()
	if err != nil {
		return errors.Wrap(err, "write registry config")
	}

	logrus.Info("starting registry container")
	container := &v1.Container{
		ID:    registryID,
		Image: "docker.io/library/registry:2.7.1",
		Configs: []*v1.ConfigFile{
			{
				ID:   registryConfigFilename,
				Path: "/etc/docker/registry/config.yml",
			},
		},
	}
	any, err := typeurl.MarshalAny(&v1.HostNetwork{})
	if err != nil {
		panic(err)
	}
	container.Networks = append(container.Networks, any)

	container.Resources = &v1.Resources{
		NoFile: 2048,
		Cpus:   2.0,
		Memory: 512,
	}
	if _, err := r.orbit.Create(ctx, &v1.CreateRequest{
		Container: container,
	}); err != nil {
		return errors.Wrap(err, "create registry container")
	}
	return nil
}

type prometheusContainer struct {
	orbit *util.LocalAgent
	ip    net.IP
}

const prometheusConfig = `global:
  scrape_interval:     30s # By default, scrape targets every 15 seconds.

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['%s:9090']`

const prometheusConfigFilename = "controller-prometheus.yml"

func (r *prometheusContainer) Start(ctx context.Context) error {
	if _, err := r.orbit.Get(ctx, &v1.GetRequest{
		ID: promID,
	}); err == nil {
		logrus.Debug("existing prometheus container is running")
		return nil
	}

	paths := opts.Paths{
		Cluster: ClusterFS,
	}
	f, err := os.Create(paths.ConfigPath(prometheusConfigFilename))
	if err != nil {
		return errors.Wrap(err, "create prometheus config")
	}
	_, err = fmt.Fprintf(f, prometheusConfig, r.ip.To4().String())
	f.Close()
	if err != nil {
		return errors.Wrap(err, "write prometheus config")
	}

	logrus.Info("starting prometheus container")
	container := &v1.Container{
		ID:    promID,
		Image: "docker.io/prom/prometheus:v2.9.2",
		Process: &v1.Process{
			Args: []string{
				fmt.Sprintf("--web.listen-address=%s:9090", r.ip.To4().String()),
				"--storage.tsdb.retention=15d",
				"--storage.tsdb.min-block-duration=30m",
				"--storage.tsdb.max-block-duration=1h",
				"--config.file=/etc/prometheus/prometheus.yml",
			},
		},
		Configs: []*v1.ConfigFile{
			{
				ID:   prometheusConfigFilename,
				Path: "/etc/prometheus/prometheus.yml",
			},
		},
	}
	any, err := typeurl.MarshalAny(&v1.HostNetwork{})
	if err != nil {
		panic(err)
	}
	container.Networks = append(container.Networks, any)

	container.Resources = &v1.Resources{
		NoFile: 2048,
		Cpus:   2.0,
		Memory: 2048,
	}
	if _, err := r.orbit.Create(ctx, &v1.CreateRequest{
		Container: container,
	}); err != nil {
		return errors.Wrap(err, "create prometheus container")
	}
	return nil
}
