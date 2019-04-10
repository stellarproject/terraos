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
	"path/filepath"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/diff/apply"
	"github.com/containerd/containerd/rootfs"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/containerd/snapshots/overlay"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func newSnapshotter(root string) (snapshots.Snapshotter, error) {
	root = filepath.Join(root, "overlay")
	return overlay.NewSnapshotter(root, overlay.AsynchronousRemove)
}

func unpackSnapshots(ctx context.Context, store content.Store, sn snapshots.Snapshotter, desc *v1.Descriptor) (string, error) {
	applier := apply.NewFileSystemApplier(store)
	_, layers, err := getLayers(ctx, store, *desc)
	if err != nil {
		return "", err
	}
	chain, err := rootfs.ApplyLayers(ctx, layers, sn, applier)
	if err != nil {
		return "", err
	}
	return chain.String(), nil
}
