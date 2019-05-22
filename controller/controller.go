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
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/v1"
	"github.com/stellarproject/terraos/config"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/disk"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/iscsi"
	"github.com/stellarproject/terraos/pkg/pxe"
	"github.com/stellarproject/terraos/pkg/resolvconf"
	"github.com/stellarproject/terraos/util"
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
	KeyPXEVersion   = "stellarproject.io/controller/pxe/version"
)

type IPType int

const (
	ISCSI IPType = iota + 1
	Management
	Gateway
	TFTP
	Orbit
)

var empty = &types.Empty{}

type infraContainer interface {
	Start(context.Context) error
}

func New(client *containerd.Client, ipConfig map[IPType]net.IP, pool *redis.Pool, orbit *util.LocalAgent) (*Controller, error) {
	if err := btrfs.Check(); err != nil {
		return nil, err
	}
	if err := iscsi.Check(); err != nil {
		return nil, err
	}
	if err := startContainers(orbit, ipConfig[Management]); err != nil {
		return nil, errors.Wrap(err, "start containers")
	}
	for _, p := range []string{ClusterFS, ISCSIPath, TFTPPath} {
		if err := os.MkdirAll(p, 0755); err != nil {
			return nil, errors.Wrapf(err, "mkdir %s", p)
		}
	}
	return &Controller{
		ips:    ipConfig,
		client: client,
		orbit:  orbit,
		pool:   pool,
		kernel: "/vmlinuz",
		initrd: "/initrd.img",
	}, nil
}

func startContainers(orbit *util.LocalAgent, ip net.IP) error {
	ctx := namespaces.WithNamespace(context.Background(), config.DefaultNamespace)
	containers := []infraContainer{
		&redisContainer{orbit: orbit, ip: ip},
		&registryContainer{orbit: orbit, ip: ip},
		&prometheusContainer{orbit: orbit, ip: ip},
	}
	for _, c := range containers {
		if err := c.Start(ctx); err != nil {
			return errors.Wrap(err, "start container")
		}
	}
	return nil
}

type Controller struct {
	mu sync.Mutex

	client *containerd.Client
	pool   *redis.Pool
	orbit  *util.LocalAgent

	ips    map[IPType]net.IP
	kernel string
	initrd string
}

func (c *Controller) Close() error {
	logrus.Debug("closing controller")
	err := c.pool.Close()
	if oerr := c.orbit.Close(); err == nil {
		err = oerr
	}
	if cerr := c.client.Close(); err == nil {
		err = cerr
	}
	return err
}

func (c *Controller) Info(ctx context.Context, _ *types.Empty) (*v1.InfoResponse, error) {
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	version, err := redis.String(conn.Do("GET", KeyPXEVersion))
	if err != nil {
		return nil, errors.Wrap(err, "get pxe version")
	}
	resp := &v1.InfoResponse{
		PxeVersion: version,
		Gateway:    c.ips[Gateway].To4().String(),
	}

	return resp, nil
}

func (c *Controller) List(ctx context.Context, _ *types.Empty) (*v1.ListNodeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logrus.Debug("listing nodes")
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	nodes, err := redis.ByteSlices(conn.Do("HVALS", KeyNodes))
	if err != nil {
		return nil, errors.Wrap(err, "get all nodes from store")
	}
	var resp v1.ListNodeResponse
	for _, data := range nodes {
		var node v1.Node
		if err := proto.Unmarshal(data, &node); err != nil {
			return nil, errors.Wrap(err, "unmarshal node")
		}
		resp.Nodes = append(resp.Nodes, &node)
	}
	return &resp, nil
}

func (c *Controller) get(ctx context.Context, hostname string) (*v1.Node, error) {
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("HGET", KeyNodes, hostname))
	if err != nil {
		return nil, errors.Wrapf(err, "get node %s", hostname)
	}
	var node v1.Node
	if err := proto.Unmarshal(data, &node); err != nil {
		return nil, errors.Wrap(err, "unmarshal node")
	}
	return &node, nil
}

func (c *Controller) Delete(ctx context.Context, r *v1.DeleteNodeRequest) (*types.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hostname := r.Hostname
	logrus.WithField("node", hostname).Info("deleting node")
	node, err := c.get(ctx, hostname)
	if err != nil {
		return nil, errors.Wrap(err, "get node information")
	}

	uri, err := url.Parse(node.Fs.BackingUri)
	if err != nil {
		return nil, errors.Wrap(err, "parse backing uri")
	}
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	if uri.Scheme == "iscsi" {
		var (
			target = iscsi.LoadTarget(iscsi.IQN(node.TargetIqn), int(node.TargetID))
			lun    = iscsi.LoadLun(int(node.LunID), node.LunPath, node.Fs.FsSize)
		)
		if err := target.Delete(ctx, lun); err != nil {
			return nil, errors.Wrap(err, "delete target and lun from tgt")
		}
		if err := lun.Delete(); err != nil {
			return nil, errors.Wrap(err, "delete lun file")
		}

		/*
			KeyTarget       = "stellarproject.io/controller/target/%d"
			KeyLUN          = "stellarproject.io/controller/lun/%d"
			KeyTargetLunIDs = "stellarproject.io/controller/target/%d/ids"
			KeyTargetLuns   = "stellarproject.io/controller/target/%d/luns"
		*/
		if _, err := conn.Do("DEL", fmt.Sprintf(KeyTarget, target.ID())); err != nil {
			return nil, errors.Wrap(err, "delete target from kv")
		}
		if _, err := conn.Do("DEL", fmt.Sprintf(KeyTargetLuns, lun.ID())); err != nil {
			return nil, errors.Wrap(err, "delete lun from kv")
		}
		if _, err := conn.Do("DEL", fmt.Sprintf(KeyTargetLuns, target.ID())); err != nil {
			return nil, errors.Wrap(err, "delete target luns from kv")
		}
		if _, err := conn.Do("DEL", fmt.Sprintf(KeyTargetLunIDs, target.ID())); err != nil {
			return nil, errors.Wrap(err, "delete target luns ids from kv")
		}
	}
	if _, err := conn.Do("HDEL", KeyNodes, hostname); err != nil {
		return nil, errors.Wrap(err, "delete node from kv")
	}

	p := &pxe.PXE{
		MAC: node.Mac,
	}
	path := filepath.Join(TFTPPath, "pxelinux.cfg", p.Filename())
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "delete pxe config")
		}
	}
	return empty, nil
}

func (c *Controller) InstallPXE(ctx context.Context, r *v1.InstallPXERequest) (*types.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logrus.Debug("installing new pxe image")

	ctx = namespaces.WithNamespace(ctx, "controller")
	repo := image.Repo(r.Image)
	if repo == "" {
		return nil, errors.New("no pxe image specified")
	}
	log := logrus.WithField("image", repo)

	log.Infof("installing pxe version %s", repo.Version())

	log.Debug("fetching image")
	i, err := c.client.Fetch(ctx, r.Image)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch pxe image %s", r.Image)
	}
	log.Debug("unpacking image")
	if err := image.Unpack(ctx, c.client.ContentStore(), &i.Target, "/"); err != nil {
		return nil, errors.Wrap(err, "unpack pxe image")
	}
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	if _, err := conn.Do("SET", KeyPXEVersion, repo.Version()); err != nil {
		return nil, errors.Wrap(err, "set pxe version")
	}
	return empty, nil
}

func (c *Controller) Provision(ctx context.Context, r *v1.ProvisionNodeRequest) (*v1.ProvisionNodeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	log := logrus.WithField("node", r.Hostname)
	log.Info("provisioning new node")

	node := &v1.Node{
		Hostname: r.Hostname,
		Mac:      r.Mac,
		Image:    r.Image,
		Fs:       r.Fs,
	}
	// do the initial save so we know this host does not exist
	if err := c.saveNode(node); err != nil {
		return nil, err
	}
	ctx = namespaces.WithNamespace(ctx, "controller")
	log.Debug("provision disk")
	if err := c.provisionDisk(ctx, node); err != nil {
		return nil, errors.Wrap(err, "provision node disk")
	}
	log.Debug("save node information")
	if err := c.updateNode(node); err != nil {
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
	log := logrus.WithField("node", node.Hostname)
	iqn := iscsi.NewIQN(2024, "san.crosbymichael.com", node.Hostname, 0)
	log.Infof("created new iqn %s", iqn)
	lun, err := iscsi.NewLun(ctx, filepath.Join(ISCSIPath, fmt.Sprintf("%s.lun", node.Hostname)), node.Fs.FsSize)
	if err != nil {
		return "", errors.Wrap(err, "create lun")
	}
	log.Debug("installing image")
	if err := c.installImage(ctx, node, uri, lun); err != nil {
		return "", errors.Wrap(err, "install image onto lun")
	}
	targetID, err := c.getNextTargetID()
	if err != nil {
		return "", err
	}
	log.Infof("creating new target with id %d", targetID)
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
	log.Infof("creating lun with id %d", lid)
	if err := target.Attach(ctx, lun, lid); err != nil {
		return "", errors.Wrap(err, "attach lun to target")
	}
	if err := c.saveLUN(lun); err != nil {
		return "", err
	}
	if err := c.saveTargetAndLun(target, lun); err != nil {
		return "", err
	}
	node.TargetID = int64(target.ID())
	node.LunID = int64(lun.ID())
	node.LunPath = lun.Path()
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
	files := []disk.File{
		&resolveConfFile{
			conf: &resolvconf.Conf{
				Nameservers: []string{
					c.ips[Gateway].To4().String(),
				},
			},
		},
	}
	if err := d.Write(ctx, image.Repo(node.Image), c.client.ContentStore(), files); err != nil {
		d.Unmount(ctx)
		return errors.Wrap(err, "write image to disk")
	}
	if err := d.Unmount(ctx); err != nil {
		return errors.Wrap(err, "unmount disk")
	}
	return nil
}

type resolveConfFile struct {
	conf *resolvconf.Conf
}

func (c *resolveConfFile) Write(path string) error {
	path = filepath.Join(path, resolvconf.DefaultPath)
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "create resolv.conf file %s", path)
	}
	defer f.Close()
	return c.conf.Write(f)
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

func (c *Controller) updateNode(node *v1.Node) error {
	conn := c.pool.Get()
	defer conn.Close()
	data, err := proto.Marshal(node)
	if err != nil {
		return errors.Wrap(err, "marshal node")
	}
	if _, err := conn.Do("HSET", KeyNodes, node.Hostname, data); err != nil {
		return errors.Wrapf(err, "update node %s", node.Hostname)
	}
	return nil
}
