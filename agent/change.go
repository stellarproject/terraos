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

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	v1 "github.com/stellarproject/terraos/api/v1/orbit"
	"github.com/stellarproject/terraos/opts"
	"github.com/stellarproject/terraos/pkg/flux"
)

type change interface {
	update(context.Context, containerd.Container) error
}

type imageUpdateChange struct {
	ref    string
	client *containerd.Client
}

func (c *imageUpdateChange) update(ctx context.Context, container containerd.Container) error {
	image, err := c.client.Pull(ctx, c.ref, containerd.WithPullUnpack, withPlainRemote(c.ref))
	if err != nil {
		return err
	}
	return container.Update(ctx, flux.WithUpgrade(image))
}

type configChange struct {
	c      *v1.Container
	client *containerd.Client
	config *Config
}

func (c *configChange) update(ctx context.Context, container containerd.Container) error {
	image, err := c.client.GetImage(ctx, c.c.Image)
	if err != nil {
		return err
	}
	return container.Update(ctx, opts.WithSetPreviousConfig, opts.WithOrbitConfig(c.config.Paths(c.c.ID), c.c, image))
}

func pauseAndRun(ctx context.Context, container containerd.Container, fn func() error) error {
	task, err := container.Task(ctx, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return fn()
		}
		return err
	}
	if err := task.Pause(ctx); err != nil {
		return err
	}
	// have a resume context incase the original context gets canceled
	rctx := relayContext(context.Background())
	defer task.Resume(rctx)
	return fn()
}

func withImage(i containerd.Image) containerd.UpdateContainerOpts {
	return func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
		c.Image = i.Name()
		return nil
	}
}
