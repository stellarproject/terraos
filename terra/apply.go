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
	"io/ioutil"
	"os"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"
)

const (
	configKey   = "config"
	configRWKey = "config-rw"
)

func applyConfig(ctx context.Context, http bool, store content.Store, sn snapshots.Snapshotter, name string) error {
	configDesc, err := fetch(ctx, http, store, name)
	if err != nil {
		return err
	}
	tmp, err := ioutil.TempDir(disk("tmp"), "config-rw-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	mounts, err := sn.Mounts(ctx, configRWKey)
	if err != nil {
		return err
	}
	if err := mount.All(mounts, tmp); err != nil {
		return err
	}
	defer mount.UnmountAll(tmp, 0)

	return unpackFlat(ctx, store, configDesc, tmp)
}
