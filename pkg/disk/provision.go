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

	"github.com/Sirupsen/logrus"
	"github.com/containerd/containerd/content"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/iscsi"
	"github.com/stellarproject/terraos/pkg/mkfs"
)

type Disk interface {
	Format(ctx context.Context, fstype, label string) error
	Provision(context.Context, string, *v1.Node) error
	Write(context.Context, image.Repo, content.Store, []File) error
	Unmount(context.Context) error
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

func (l *localDisk) Format(ctx context.Context, fstype, label string) error {
	var args []string
	if fstype == mkfs.Btrfs {
		args = append(args, "-f")
	}
	if err := mkfs.Mkfs(fstype, l.device, label, args...); err != nil {
		return errors.Wrap(err, "mkfs device")
	}
	return nil
}

func (l *localDisk) Write(ctx context.Context, repo image.Repo, store content.Store, files []File) error {
	if err := write(ctx, repo, l.root, l.path, store, l.subvolumes, files); err != nil {
		return errors.Wrap(err, "write image to disk")
	}
	return nil
}

func (l *localDisk) Unmount(ctx context.Context) error {
	for _, p := range l.mounts {
		syscall.Unmount(p, 0)
	}
	if err := syscall.Unmount(l.root, 0); err != nil {
		logrus.WithError(err).Error("unmount disk")
		return nil
	}
	if err := os.RemoveAll(l.root); err != nil {
		return errors.Wrap(err, "remove path")
	}
	return nil
}

func (l *localDisk) Provision(ctx context.Context, fstype string, node *v1.Node) error {
	var (
		path = l.root
	)
	l.path = path
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Wrap(err, "mkdir lun mount path")
	}
	if err := syscall.Mount(l.device, path, fstype, 0, ""); err != nil {
		return errors.Wrap(err, "mount device")
	}
	var (
		repo    = image.Repo(node.Image)
		version = repo.Version()
	)
	var subvolumes []btrfs.Subvolume
	if fstype == mkfs.Btrfs {
		subvolumes = getSubVolumes(node)
		l.subvolumes = subvolumes
		if err := btrfs.CreateSubvolumes(append(subvolumes, btrfs.Subvolume{
			Name: version,
		}), path); err != nil {
			return errors.Wrap(err, "create subvolumes")
		}
		rootPath := filepath.Join(path, version)
		paths, err := btrfs.OverlaySubvolumes(rootPath, subvolumes, path)
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

func write(ctx context.Context, repo image.Repo, root, path string, store content.Store, subvolumes []btrfs.Subvolume, files []File) error {
	desc, err := image.Fetch(ctx, false, store, string(repo))
	if err != nil {
		return errors.Wrap(err, "fetch image")
	}
	if err := image.Unpack(ctx, store, desc, path); err != nil {
		return errors.Wrap(err, "unpack image")
	}
	if err := writeVersion(repo.Version(), root); err != nil {
		return errors.Wrap(err, "write version file")
	}
	if len(subvolumes) > 0 {
		if err := writeFstab(subvolumes, path); err != nil {
			return errors.Wrap(err, "write fstab file")
		}
	}
	for i, f := range files {
		if err := f.Write(path); err != nil {
			return errors.Wrapf(err, "write file %d", i)
		}
	}
	return nil
}

func NewLunDisk(lun *iscsi.Lun) Disk {
	return &lunDisk{
		lun: lun,
	}
}

type lunDisk struct {
	lun        *iscsi.Lun
	mounts     []string
	path       string
	root       string
	subvolumes []btrfs.Subvolume
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

func (l *lunDisk) Write(ctx context.Context, repo image.Repo, store content.Store, files []File) error {
	if err := write(ctx, repo, l.root, l.path, store, l.subvolumes, files); err != nil {
		return errors.Wrap(err, "write image to disk")
	}
	return nil
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
		subvolumes = getSubVolumes(node)
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

func writeFstab(subvolumes []btrfs.Subvolume, path string) error {
	f, err := os.Create(filepath.Join(path, "/etc/fstab"))
	if err != nil {
		return errors.Wrap(err, "create fstab")
	}
	defer f.Close()
	var entries []*fstab.Entry
	for _, s := range subvolumes {
		entries = append(entries, &fstab.Entry{
			Type:   mkfs.Btrfs,
			Device: "LABEL=os",
			Path:   s.Path,
			Pass:   2,
			Options: []string{
				fmt.Sprintf("subvol=/%s", s.Name),
			},
		})
	}
	if err := fstab.Write(f, entries); err != nil {
		return errors.Wrap(err, "write fstab")
	}
	return nil
}

func getSubVolumes(node *v1.Node) []btrfs.Subvolume {
	subvolumes := []btrfs.Subvolume{
		{
			Name: "home",
			Path: "/home",
		},
		{
			Name: "containerd",
			Path: "/var/lib/containerd",
		},
		{
			Name: "log",
			Path: "/var/log",
		},
	}
	for _, s := range node.Fs.Subvolumes {
		subvolumes = append(subvolumes, btrfs.Subvolume{
			Name: s.Name,
			Path: s.Path,
		})
	}
	return subvolumes
}
