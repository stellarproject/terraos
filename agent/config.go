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

package agent

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/containerd/containerd"
	"github.com/stellarproject/terraos/opts"
	"github.com/stellarproject/terraos/util"
)

type Config struct {
	ID           string        `toml:"id"`
	Domain       string        `toml:"domain,omitempty"`
	State        string        `toml:"state"`
	Iface        string        `toml:"iface"`
	PlainRemotes []string      `toml:"plain_remotes"`
	Interval     time.Duration `toml:"interval"`
	ClusterDir   string        `toml:"cluster_dir"`

	ip    string
	ipErr error
	ipO   sync.Once
}

func (c *Config) IP() (string, error) {
	c.ipO.Do(func() {
		c.ip, c.ipErr = util.GetIP(c.Iface)
	})
	if c.ipErr != nil {
		return "", c.ipErr
	}
	return c.ip, nil
}

func (c *Config) Paths(id string) opts.Paths {
	return opts.Paths{
		State:   filepath.Join(c.State, id),
		Cluster: c.ClusterDir,
	}
}

type host struct {
	ip string
}

func (n *host) Create(_ context.Context, _ containerd.Container) (string, error) {
	return n.ip, nil
}

func (n *host) Remove(_ context.Context, _ containerd.Container) error {
	return nil
}

type none struct {
}

func (n *none) Create(_ context.Context, _ containerd.Container) (string, error) {
	return "", nil
}

func (n *none) Remove(_ context.Context, _ containerd.Container) error {
	return nil
}
