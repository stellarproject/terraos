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
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1"
	"github.com/stellarproject/terraos/pkg/disk"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/iscsi"
	"github.com/stellarproject/terraos/pkg/pxe"
)

const (
	ClusterFS = "/cluster"
	ISCSIPath = "/iscsi"
	TFTPPath  = "/tftp"

	KeyTargetIDs    = "stellarproject.io/controller/target/ids"
	KeyTarget       = "stellarproject.io/controller/target/%d"
	KeyLUN          = "stellarproject.io/controller/lun/%d"
	KeyTargetLunIDs = "stellarproject.io/controller/target/%d/ids"
	KeyTargetLuns   = "stellarproject.io/controller/target/%d/luns"
	KeyNodes        = "stellarproject.io/controller/nodes"
)

type IPType int

const (
	ISCSI IPType = iota + 1
	Management
	Gateway
	TFTP
)

var empty = &types.Empty{}

func New(client *containerd.Client, ipConfig map[IPType]net.IP, pool *redis.Pool) (*Controller, error) {
	return &Controller{
		ips:    ipConfig,
		client: client,
		kernel: "/vmlinuz",
		initrd: "/initrd.img",
	}, nil
}

type Controller struct {
	mu sync.Mutex

	ips    map[IPType]net.IP
	client *containerd.Client
	pool   *redis.Pool
	kernel string
	initrd string
}

func (c *Controller) Close() error {
	return c.pool.Close()
}

func (c *Controller) Get(ctx context.Context, r *v1.GetNodeRequest) (*v1.GetNodeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("HGET", KeyNodes, r.Hostname))
	if err != nil {
		return nil, errors.Wrapf(err, "get node %s", r.Hostname)
	}
	var node v1.Node
	if err := proto.Unmarshal(data, &node); err != nil {
		return nil, errors.Wrap(err, "unmarshal node")
	}
	return &v1.GetNodeResponse{
		Node: &node,
	}, nil
}

func (c *Controller) InstallPXE(ctx context.Context, r *v1.InstallPXERequest) (*types.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx = namespaces.WithNamespace(ctx, "controller")
	i, err := c.client.Fetch(ctx, r.Image)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch pxe image %s", r.Image)
	}
	if err := image.Unpack(ctx, c.client.ContentStore(), &i.Target, "/"); err != nil {
		return nil, errors.Wrap(err, "unpack pxe image")
	}
	return empty, nil
}

func (c *Controller) Provision(ctx context.Context, r *v1.ProvisionNodeRequest) (*v1.ProvisionNodeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.Mkdir(filepath.Join(ClusterFS, r.Hostname), 0755); err != nil {
		return nil, errors.Wrap(err, "mkdir node directory")
	}
	node := &v1.Node{
		Hostname: r.Hostname,
		Mac:      r.Mac,
		Image:    r.Image,
		Fs:       r.Fs,
	}
	ctx = namespaces.WithNamespace(ctx, "controller")
	if err := c.provisionDisk(ctx, node); err != nil {
		return nil, errors.Wrap(err, "provision node disk")
	}
	if err := c.saveNode(node); err != nil {
		return nil, err
	}
	return &v1.ProvisionNodeResponse{
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
	case "local":
		// node will have a manually installed local disk, nothing for us to do
	default:
		return errors.Wrapf(err, "invalid backing uri %s", node.Fs.BackingUri)
	}
	return nil
}

// TODO: parallel create lun and fetch image
func (c *Controller) createISCSI(ctx context.Context, node *v1.Node, uri *url.URL) (iscsi.IQN, error) {
	iqn := iscsi.NewIQN(2024, "san.crosbymichael.com", node.Hostname, 0)
	lun, err := iscsi.NewLun(ctx, filepath.Join(ISCSIPath, fmt.Sprintf("%s.lun", node.Hostname)), node.Fs.FsSize)
	if err != nil {
		return "", errors.Wrap(err, "create lun")
	}
	if err := c.installImage(ctx, node, uri, lun); err != nil {
		return "", errors.Wrap(err, "install image onto lun")
	}
	targetID, err := c.getNextTargetID()
	if err != nil {
		return "", err
	}
	target, err := iscsi.NewTarget(ctx, iqn, targetID)
	if err != nil {
		return "", errors.Wrapf(err, "create target %s", iqn)
	}
	if err := c.saveTarget(target); err != nil {
		return "", err
	}
	if err := target.AcceptAllInitiators(ctx); err != nil {
		return "", errors.Wrap(err, "accept all initiators for target")
	}
	lid, err := c.getNextTargetLUNID(target)
	if err != nil {
		return "", err
	}
	if err := target.Attach(ctx, lun, lid); err != nil {
		return "", errors.Wrap(err, "attach lun to target")
	}
	if err := c.saveLUN(lun); err != nil {
		return "", err
	}
	if err := c.saveTargetAndLun(target, lun); err != nil {
		return "", err
	}
	return iqn, nil
}

func (c *Controller) installImage(ctx context.Context, node *v1.Node, uri *url.URL, lun *iscsi.Lun) error {
	var (
		t = uri.Host
		d = disk.NewLunDisk(lun)
	)
	if err := d.Format(ctx, t, "os"); err != nil {
		return errors.Wrap(err, "format disk")
	}
	if err := d.Provision(ctx, t, node); err != nil {
		d.Unmount(ctx)
		return errors.Wrap(err, "provision disk")
	}
	// TODO: write resolv.conf
	if err := d.Write(ctx, image.Repo(node.Image), c.client.ContentStore()); err != nil {
		d.Unmount(ctx)
		return errors.Wrap(err, "write image to disk")
	}
	if err := d.Unmount(ctx); err != nil {
		return errors.Wrap(err, "unmount disk")
	}
	return nil
}

func (c *Controller) writePXEConfig(node *v1.Node, version string) error {
	p := &pxe.PXE{
		Default:      "terra",
		MAC:          node.Mac,
		InitiatorIQN: node.InitiatorIqn,
		TargetIQN:    node.TargetIqn,
		TargetIP:     c.ips[ISCSI].To4().String(),
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

func (c *Controller) getNextTargetID() (int, error) {
	conn := c.pool.Get()
	defer conn.Close()

	id, err := redis.Int(conn.Do("INCR", KeyTargetIDs))
	if err != nil {
		return -1, errors.Wrap(err, "get next target id")
	}
	return id, nil
}

func (c *Controller) getNextTargetLUNID(t *iscsi.Target) (int, error) {
	conn := c.pool.Get()
	defer conn.Close()

	id, err := redis.Int(conn.Do("INCR", fmt.Sprintf(KeyTargetLunIDs, t.ID())))
	if err != nil {
		return -1, errors.Wrap(err, "get next target lun id")
	}
	return id, nil
}

func (c *Controller) saveTarget(t *iscsi.Target) error {
	conn := c.pool.Get()
	defer conn.Close()
	args := []interface{}{
		"iqn", string(t.IQN()),
	}
	if _, err := conn.Do("HMSET", append([]interface{}{fmt.Sprintf(KeyTarget, t.ID())}, args...)...); err != nil {
		return errors.Wrap(err, "set target hash")
	}
	return nil
}

func (c *Controller) saveLUN(t *iscsi.Lun) error {
	conn := c.pool.Get()
	defer conn.Close()
	args := []interface{}{
		"size", t.Size(),
		"path", t.Path(),
	}
	if _, err := conn.Do("HMSET", append([]interface{}{fmt.Sprintf(KeyLUN, t.ID())}, args...)...); err != nil {
		return errors.Wrap(err, "set lun hash")
	}
	return nil
}

func (c *Controller) saveTargetAndLun(t *iscsi.Target, lun *iscsi.Lun) error {
	conn := c.pool.Get()
	defer conn.Close()
	if _, err := conn.Do("SADD", fmt.Sprintf(KeyTargetLuns, t.ID()), lun.ID()); err != nil {
		return errors.Wrap(err, "set target lun")
	}
	return nil
}

func (c *Controller) saveNode(node *v1.Node) error {
	conn := c.pool.Get()
	defer conn.Close()
	data, err := proto.Marshal(node)
	if err != nil {
		return errors.Wrap(err, "marshal node")
	}
	if _, err := conn.Do("HSETNX", KeyNodes, node.Hostname, data); err != nil {
		return errors.Wrapf(err, "save node %s", node.Hostname)
	}
	return nil
}
