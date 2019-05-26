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

package disk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1/types"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/iscsi"
	"github.com/stellarproject/terraos/pkg/mkfs"
)

type Disk interface {
	Write(context.Context, image.Repo, content.Store, []File) error
}

type File interface {
	Write(root string) error
}

func NewLocalDisk(device string) Disk {
	return &localDisk{
		device: device,
		root:   "/sd",
	}
}

type localDisk struct {
	device     string
	root       string
	path       string
	mounts     []string
	subvolumes []btrfs.Subvolume
}

func (l *localDisk) Write(ctx context.Context, repo image.Repo, store content.Store, files []File) error {
	desc, err := image.Fetch(ctx, false, store, string(repo))
	if err != nil {
		return errors.Wrap(err, "fetch image")
	}
	if err := image.Unpack(ctx, store, desc, l.path); err != nil {
		return errors.Wrap(err, "unpack image")
	}
	if err := writeVersion(repo.Version(), l.root); err != nil {
		return errors.Wrap(err, "write version file")
	}
	if len(l.subvolumes) > 0 {
		if err := writeFstab(l.subvolumes, l.path); err != nil {
			return errors.Wrap(err, "write fstab file")
		}
	}
	for i, f := range files {
		if err := f.Write(l.path); err != nil {
			return errors.Wrapf(err, "write file %d", i)
		}
	}
	return nil
}

func NewLunDisk(lun *iscsi.Lun, client *containerd.Client) Disk {
	return &lunDisk{
		lun:    lun,
		client: client,
	}
}

type lunDisk struct {
	lun        *iscsi.Lun
	mounts     []string
	path       string
	root       string
	subvolumes []btrfs.Subvolume
	client     *containerd.Client
}

func (l *lunDisk) Unmount(ctx context.Context) error {
	for _, p := range l.mounts {
		syscall.Unmount(p, 0)
	}
	if err := syscall.Unmount(l.root, 0); err != nil {
		return errors.Wrap(err, "unmount disk")
	}
	if err := os.RemoveAll(l.root); err != nil {
		return errors.Wrap(err, "remove path")
	}
	return nil
}

func (l *lunDisk) Write(ctx context.Context, repo image.Repo, _ content.Store, files []File) error {
	img, err := l.fetch(ctx, repo)
	if err != nil {
		return errors.Wrap(err, "fetch image")
	}
	desc := img.Target()
	if err := image.Unpack(ctx, l.client.ContentStore(), &desc, l.path); err != nil {
		return errors.Wrap(err, "unpack image")
	}
	if err := writeVersion(repo.Version(), l.root); err != nil {
		return errors.Wrap(err, "write version file")
	}
	if len(l.subvolumes) > 0 {
		if err := writeFstab(l.subvolumes, l.path); err != nil {
			return errors.Wrap(err, "write fstab file")
		}
	}
	for i, f := range files {
		if err := f.Write(l.path); err != nil {
			return errors.Wrapf(err, "write file %d", i)
		}
	}
	return nil
}

func (l *lunDisk) fetch(ctx context.Context, repo image.Repo) (containerd.Image, error) {
	image, err := l.client.GetImage(ctx, string(repo))
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, errors.Wrapf(err, "image get error %s", repo)
		}
		i, err := l.client.Fetch(ctx, string(repo))
		if err != nil {
			return nil, errors.Wrapf(err, "fetch repo %s", repo)
		}
		image = containerd.NewImage(l.client, i)
	}
	return image, nil
}

func (l *lunDisk) Format(ctx context.Context, fstype, label string) error {
	var args []string
	if fstype == mkfs.Btrfs {
		args = append(args, "-f")
	}
	if err := mkfs.Mkfs(fstype, l.lun.Path(), label, args...); err != nil {
		return errors.Wrap(err, "mkfs of lun")
	}
	return nil
}

func (l *lunDisk) Provision(ctx context.Context, fstype string, node *v1.Node) error {
	var (
		path = l.lun.Path() + ".mnt"
		root = path
	)
	l.root = root
	l.path = path
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Wrap(err, "mkdir lun mount path")
	}
	if err := l.lun.LocalMount(ctx, fstype, path); err != nil {
		return errors.Wrap(err, "mount lun")
	}
	var (
		repo    = image.Repo(node.Image)
		version = repo.Version()
	)
	var subvolumes []btrfs.Subvolume
	if fstype == mkfs.Btrfs {
		l.subvolumes = subvolumes
		if err := btrfs.CreateSubvolumes(append(subvolumes, btrfs.Subvolume{
			Name: version,
		}), root); err != nil {
			return errors.Wrap(err, "create subvolumes")
		}
		rootPath := filepath.Join(path, version)
		paths, err := btrfs.OverlaySubvolumes(rootPath, subvolumes, root)
		if err != nil {
			return errors.Wrap(err, "overlay subvolumes")
		}
		l.mounts = paths
		// set path to the rootpath so that the image is unpacked over the top
		// of the version and subvolumes
		path = rootPath
	}
	l.path = path
	return nil
}

func writeVersion(version, root string) error {
	path := filepath.Join(root, "VERSION")
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "create version file %s", path)
	}
	defer f.Close()
	if _, err := fmt.Fprint(f, version); err != nil {
		return errors.Wrapf(err, "write version to file %s", path)
	}
	return nil
}
