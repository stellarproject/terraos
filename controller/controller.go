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
	"hash/fnv"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"syscall"

	"github.com/containerd/containerd/content"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/iscsi"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/stellarproject/terraos/pkg/pxe"
)

const (
	ClusterFS        = "/cluster"
	ContentStorePath = "/content-store"
	ISCSIPath        = "/iscsi"
	TFTPPath         = "/tftp"
)

func New(ip net.IP) (*Controller, error) {
	store, err := image.NewContentStore(ContentStorePath)
	if err != nil {
		return errors.Wrap(err, "create content store")
	}
	return &Controller{
		ip:     ip,
		store:  store,
		kernel: "/vmlinuz",
		initrd: "/initrd.img",
	}, nil
}

type Controller struct {
	ip     net.IP
	store  content.Store
	kernel string
	initrd string
}

func (c *Controller) Close() error {
	return nil
}

func (c *Controller) RegisterNode(ctx context.Context, r *v1.RegisterNodeRequest) (*v1.RegisterNodeResponse, error) {
	if err := os.Mkdir(filepath.Join(ClusterFS, r.Hostname), 0755); err != nil {
		return nil, errors.Wrap(err, "mkdir node directory")
	}
	node := &v1.Node{
		Hostname: r.Hostname,
		Mac:      r.Mac,
		Image:    r.Image,
		Fs:       r.Fs,
	}
	if err := c.provisionDisk(ctx, node); err != nil {
		return errors.Wrap(err, "provision node disk")
	}
	return &v1.RegisterNodeResponse{
		Node: node,
	}, nil
}

func (c *Controller) provisionDisk(ctx context.Context, node *v1.Node) error {
	if node.Fs == nil {
		return errors.New("no filesytem information provided for node")
	}
	uri, err := url.Parse(node.Fs.BackingUri)
	if err != nil {
		return errors.Wrapf(err, "unable to parse backing uri %s", node.Fs.BackingUri)
	}
	switch uri.Scheme {
	case "iscsi":
		repo := image.Repo(node.Image)
		version := repo.Version()
		iqn, err := c.createISCSI(ctx, node, uri)
		if err != nil {
			return errors.Wrap(err, "create iscsi disk")
		}
		initIqn := iscsi.NewIQN(2024, "node.crosbymichael.com", node.Hostname, -1)
		node.InitiatorIqn = string(initIqn)
		node.TargetIqn = string(iqn)
		if err := c.writePXEConfig(node, version); err != nil {
			return errors.Wrap(err, "write pxe config")
		}
		return nil
	case "local":
		// node will have a manually installed local disk, nothing for us to do
		return nil
	default:
		return errors.Wrapf(err, "invalid backing uri %s", node.Fs.BackingUri)
	}
	return nil
}

// TODO: parallel create lun and fetch image
func (c *Controller) createISCSI(ctx context.Context, node *v1.Node, uri *url.URL) (IQN, error) {
	iqn := iscsi.NewIQN(2024, "san.crosbymichael.com", node.Hostname, 0)
	lun, err := iscsi.NewLun(ctx, filepath.Join(ISCSIPath, fmt.Sprintf("%s.lun", node.Hostname)), node.Fs.FsSize)
	if err != nil {
		return "", errors.Wrap(err, "create lun")
	}
	if err := c.installImage(ctx, node, uri, lun); err != nil {
		return "", errors.Wrap(err, "install image onto lun")
	}
	target, err := iscsi.NewTarget(ctx, iqn, getTargetID(iqn))
	if err != nil {
		return "", errors.Wrapf(err, "create target %s", iqn)
	}
	if err := target.AcceptAllInitiators(ctx); err != nil {
		return "", errors.Wrap(err, "accept all initiators for target")
	}
	if err := target.Attach(ctx, lun); err != nil {
		return "", errors.Wrap(err, "attach lun to target")
	}
	return iqn, nil
}

func (c *Controller) installImage(ctx context.Context, node *v1.Node, uri *url.URL, lun *iscsi.Lun) error {
	t := mkfs.Type(uri.Host)
	if err := mkfs.Mkfs(t, "os", lun.Path(), "-f"); err != nil {
		return errors.Wrap(err, "mkfs of lun")
	}
	var (
		path = lun.Path() + ".mnt"
		root = path
	)
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Wrap(err, "mkdir lun mount path")
	}
	defer os.RemoveAll(path)
	if err := lun.LocalMount(ctx, t, path); err != nil {
		return errors.Wrap(err, "mount lun")
	}
	defer syscall.Unmount(path, 0)

	var (
		repo    = image.Repo(node.Image)
		version = repo.Version()
	)
	desc, err := image.Fetch(ctx, false, c.store, string(repo))
	if err != nil {
		return errors.Wrap(err, "fetch image")
	}
	var subvolumes []btrfs.Subvolume
	if t == mkfs.Btrfs {
		subvolumes = getSubVolumes(node)
		if err := btrfs.CreateSubvolumes(append(subvolumes, btrfs.Subvolume{
			Name: version,
		}), path); err != nil {
			return errors.Wrap(err, "create subvolumes")
		}
		rootPath := filepath.Join(path, version)
		paths, err := btrfs.OverlaySubvolumes(path, subvolumes, rootPath)
		if err != nil {
			return errors.Wrap(err, "overlay subvolumes")
		}
		defer func() {
			for _, p := range paths {
				syscall.Unmount(p, 0)
			}
		}()
		// set path to the rootpath so that the image is unpacked over the top
		// of the version and subvolumes
		path = rootPath
	}
	if err := image.Unpack(ctx, c.store, desc, path); err != nil {
		return errors.Wrap(err, "unpack image")
	}
	if err := writeVersion(version, root); err != nil {
		return errors.Wrap(err, "write version file")
	}
	if len(subvolumes) > 0 {
		if err := writeFstab(subvolumes, path); err != nil {
			return errors.Wrap(err, "write fstab file")
		}
	}
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

func (c *Controller) writePXEConfig(node *v1.Node, version string) error {
	p := &pxe.PXE{
		Defalt:       "terra",
		MAC:          node.Mac,
		InitiatorIQN: node.InitiatorIqn,
		TargetIQN:    node.TargetIqn,
		TargetIP:     c.ip.To4().String(),
		IP:           pxe.DHCP,
		Entries: []pxe.Entry{
			{
				Root:   "LABEL=os",
				Label:  "terra",
				Boot:   "terra",
				Kernel: c.kernel,
				Initrd: c.initrd,
				Append: []string{
					"version=" + version,
					"disk_label=os",
				},
			},
		},
	}
	path := filepath.Join(TFTPPath, "pxelinux.cfg", p.Filename())
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "create pxe file %s", path)
	}
	defer f.Close()
	if err := p.Write(f); err != nil {
		return errors.Wrap(err, "write pxe config")
	}
	return nil
}

func getTargetID(iqn iscsi.IQN) int {
	h := fnv.New32a()
	fmt.Fprint(h, iqn)
	return int(h.Sum32())
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
