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

package cni

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/containerd/containerd"
	gocni "github.com/containerd/go-cni"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/opts"
	"golang.org/x/sys/unix"
)

type Config struct {
	Type  string
	State string
	Iface string
}

func New(c Config, n gocni.CNI) (*cni, error) {
	return &cni{
		network: n,
		config:  c,
	}, nil
}

type cni struct {
	network gocni.CNI
	config  Config
}

func (n *cni) Create(ctx context.Context, task containerd.Container) (string, error) {
	path := filepath.Join(n.config.State, task.ID(), "net")
	if _, err := os.Lstat(path); err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrap(err, "lstat network namespace")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return "", errors.Wrap(err, "mkdir of network path")
		}
		if err := createNetns(path); err != nil {
			return "", errors.Wrap(err, "create netns")
		}
		result, err := n.network.Setup(task.ID(), path)
		if err != nil {
			return "", errors.Wrap(err, "setup cni network")
		}
		var ip net.IP
		for _, ipc := range result.Interfaces["eth0"].IPConfigs {
			if f := ipc.IP.To4(); f != nil {
				ip = f
				break
			}
		}
		if err := task.Update(ctx, opts.WithIP(ip.String())); err != nil {
			return "", errors.Wrap(err, "update with ip")
		}
		return ip.String(), nil
	}
	l, err := task.Labels(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get container labels for ip")
	}
	return l[opts.IPLabel], nil
}

func (n *cni) Remove(ctx context.Context, c containerd.Container) error {
	path := filepath.Join(n.config.State, c.ID(), "net")
	if err := n.network.Remove(c.ID(), path); err != nil {
		logrus.WithError(err).Error("remove cni gocni")
	}
	if err := unix.Unmount(path, 0); err != nil {
		logrus.WithError(err).Error("unmount netns")
	}
	// FIXME this could cause issues later but whatever...
	return os.RemoveAll(filepath.Dir(path))
}

func createNetns(path string) error {
	out, err := exec.Command("orbit-network", path).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(out))
	}
	return nil
}
