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
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	api "github.com/stellarproject/terraos/api/v1/services"
	v1 "github.com/stellarproject/terraos/api/v1/types"
	"github.com/stellarproject/terraos/config"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/iscsi"
	"github.com/stellarproject/terraos/pkg/pxe"
	"github.com/stellarproject/terraos/pkg/resolvconf"
	"github.com/stellarproject/terraos/stage0"
	"github.com/stellarproject/terraos/stage1"
	"github.com/stellarproject/terraos/util"
)

const (
	ClusterFS = "/cluster"
	ISCSIPath = "/iscsi"
	TFTPPath  = "/tftp"

	KeyNodes      = "stellarproject.io/controller/nodes"
	KeyPXEVersion = "stellarproject.io/controller/pxe/version"
	MaxTargetID   = 1024
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
	for _, p := range []string{
		ClusterFS,
		ISCSIPath,
		TFTPPath,
		filepath.Join(ISCSIPath, "snapshots"),
	} {
		if err := os.MkdirAll(p, 0755); err != nil {
			return nil, errors.Wrapf(err, "mkdir %s", p)
		}
	}
	c := &Controller{
		ips:    ipConfig,
		client: client,
		orbit:  orbit,
		pool:   pool,
		kernel: "/vmlinuz",
		initrd: "/initrd.img",
	}
	if err := c.restoreTargets(context.Background()); err != nil {
		return nil, errors.Wrap(err, "restore targets")
	}
	return c, nil
}

func (c *Controller) restoreTargets(ctx context.Context) error {
	nodes, err := c.nodes(ctx)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if node.InitiatorIqn == "" {
			continue
		}
		if err := c.restoreDisks(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) restoreDisks(ctx context.Context, node *v1.Node) error {
	for _, group := range node.DiskGroups {
		if group.Target != nil {
			if err := iscsi.SetTarget(ctx, group.Target); err != nil {
				if isTargetExists(err) {
					continue
				}

				return errors.Wrap(err, "set target")
			}
			for _, disk := range group.Disks {
				// > 0 means a lun disk
				if disk.ID > 0 {
					if err := iscsi.Attach(ctx, group.Target, disk); err != nil {
						return errors.Wrap(err, "attach disk to target")
					}
				}
			}
			if err := iscsi.Accept(ctx, group.Target, node.InitiatorIqn); err != nil {
				return errors.Wrap(err, "accept node iqn")
			}
			if err := iscsi.AcceptAllInitiators(ctx, group.Target); err != nil {
				return errors.Wrap(err, "accept ALL iqn")
			}
		}
	}
	return nil
}

func isTargetExists(err error) bool {
	return strings.Contains(err.Error(), "this target already exists")
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

func (c *Controller) Info(ctx context.Context, _ *types.Empty) (*api.InfoResponse, error) {
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	version, err := redis.String(conn.Do("GET", KeyPXEVersion))
	if err != nil {
		if err != redis.ErrNil {
			return nil, errors.Wrap(err, "get pxe version")
		}
	}
	resp := &api.InfoResponse{
		PxeVersion: version,
		Gateway:    c.ips[Gateway].To4().String(),
	}
	return resp, nil
}

func (c *Controller) List(ctx context.Context, _ *types.Empty) (*api.ListNodeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logrus.Debug("listing nodes")
	nodes, err := c.nodes(ctx)
	if err != nil {
		return nil, err
	}
	return &api.ListNodeResponse{
		Nodes: nodes,
	}, nil
}

func (c *Controller) nodes(ctx context.Context) ([]*v1.Node, error) {
	var out []*v1.Node
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	nodes, err := redis.ByteSlices(conn.Do("HVALS", KeyNodes))
	if err != nil {
		return nil, errors.Wrap(err, "get all nodes from store")
	}
	for _, data := range nodes {
		var node v1.Node
		if err := proto.Unmarshal(data, &node); err != nil {
			return nil, errors.Wrap(err, "unmarshal node")
		}
		out = append(out, &node)
	}
	return out, nil
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

func (c *Controller) Delete(ctx context.Context, r *api.DeleteNodeRequest) (*types.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hostname := r.Hostname
	logrus.WithField("node", hostname).Info("deleting node")
	node, err := c.get(ctx, hostname)
	if err != nil {
		return nil, errors.Wrap(err, "get node information")
	}
	return empty, c.delete(ctx, node)
}

func (c *Controller) delete(ctx context.Context, node *v1.Node) error {
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	for _, group := range node.DiskGroups {
		// TODO: muti disk support
		if group.Target == nil {
			continue
		}
		if err := iscsi.Delete(ctx, group.Target, group.Disks[0]); err != nil {
			return errors.Wrap(err, "delete target and lun from tgt")
		}
		path := filepath.Join(ISCSIPath, node.Hostname)
		if err := os.RemoveAll(path); err != nil {
			return errors.Wrap(err, "delete luns")
		}
	}
	if _, err := conn.Do("HDEL", KeyNodes, node.Hostname); err != nil {
		return errors.Wrap(err, "delete node from kv")
	}

	p := &pxe.PXE{
		MAC: node.Mac,
	}
	path := filepath.Join(TFTPPath, "pxelinux.cfg", p.Filename())
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "delete pxe config")
		}
	}
	return nil
}

func (c *Controller) InstallPXE(ctx context.Context, r *api.InstallPXERequest) (*types.Empty, error) {
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
	i, err := c.client.Fetch(ctx, r.Image, withPlainRemote(r.Image))
	if err != nil {
		return nil, errors.Wrapf(err, "fetch pxe image %s", r.Image)
	}
	log.Debug("unpacking image")
	path, err := ioutil.TempDir("", "terra-pxe-install")
	if err != nil {
		return nil, errors.Wrap(err, "create tmp pxe dir")
	}
	defer os.RemoveAll(path)

	if err := image.Unpack(ctx, c.client.ContentStore(), &i.Target, path); err != nil {
		return nil, errors.Wrap(err, "unpack pxe image")
	}
	if err := syncDir(ctx, filepath.Join(path, "tftp")+"/", TFTPPath+"/"); err != nil {
		return nil, errors.Wrap(err, "sync tftp dir")
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

func syncDir(ctx context.Context, source, target string) error {
	out, err := exec.CommandContext(ctx, "rsync", "-a", source, target).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func (c *Controller) Provision(ctx context.Context, r *api.ProvisionNodeRequest) (_ *api.ProvisionNodeResponse, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	log := logrus.WithField("node", r.Node.Hostname)
	log.Info("provisioning new node")

	node := r.Node

	image, err := c.fetch(ctx, node.Image)
	if err != nil {
		return nil, errors.Wrap(err, "fetch node image")
	}
	initIqn := iscsi.NewIQN(2024, "node.crosbymichael.com", node.Hostname, false)
	node.InitiatorIqn = string(initIqn)

	// do the initial save so we know this host does not exist
	if err := c.saveNode(node); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if err := c.delete(ctx, node); err != nil {
				log.WithError(err).Error("delete failed node")
			}
		}
	}()
	ctx = namespaces.WithNamespace(ctx, "controller")
	log.Debug("provision disk")
	if err := c.provisionTarget(ctx, node, image); err != nil {
		return nil, errors.Wrap(err, "provision node target")
	}
	if err := c.writePXEConfig(node); err != nil {
		return nil, errors.Wrap(err, "write pxe config")
	}
	log.Debug("save node information")
	if err := c.updateNode(node); err != nil {
		return nil, err
	}
	return &api.ProvisionNodeResponse{
		Node: node,
	}, nil
}

func (c *Controller) fetch(ctx context.Context, repo string) (containerd.Image, error) {
	image, err := c.client.GetImage(ctx, repo)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, errors.Wrapf(err, "image get error %s", repo)
		}
		i, err := c.client.Fetch(ctx, repo, withPlainRemote(repo))
		if err != nil {
			return nil, errors.Wrapf(err, "fetch repo %s", repo)
		}
		image = containerd.NewImage(c.client, i)
	}
	return image, nil
}

func (c *Controller) provisionTarget(ctx context.Context, node *v1.Node, image containerd.Image) (err error) {
	if len(node.DiskGroups) != 1 {
		return errors.Errorf("only 1 disk group supported with iscsi: %d groups", len(node.DiskGroups))
	}
	group := node.DiskGroups[0]
	if group.Stage == v1.Stage0 {
		return errors.New("stage0 group not supported with iscsi")
	}
	if group.GroupType != v1.Single {
		return errors.Errorf("group type %s not supported with iscsi", group.GroupType)
	}
	// todo support multiple disks
	if len(group.Disks) != 1 {
		return errors.Errorf("multiple disks not supported with iscsi: %d disks", len(group.Disks))
	}
	// TODO: create controller iqn and allow all luns
	// also add disk based lun labels for each
	target, err := c.createTarget(ctx, node)
	if err != nil {
		return errors.Wrap(err, "create target")
	}
	defer func() {
		if err != nil {
			iscsi.Delete(ctx, target, nil)
		}
	}()
	group.Target = target

	if err := c.createGroupLuns(ctx, node, group); err != nil {
		return errors.Wrapf(err, "provision disk group %s", group.Label)
	}
	defer func() {
		if err != nil {
			for _, disk := range group.Disks {
				iscsi.DeleteLun(disk)
			}
		}
	}()
	if err := stage0.Format(group); err != nil {
		return errors.Wrap(err, "format group")
	}
	if err := c.installImage(ctx, group, image); err != nil {
		return errors.Wrap(err, "install image to disk group")
	}
	for _, disk := range group.Disks {
		if err := iscsi.Attach(ctx, group.Target, disk); err != nil {
			return errors.Wrapf(err, "attach %d to target", disk.ID)
		}
	}
	return nil
}

func (c *Controller) createGroupLuns(ctx context.Context, node *v1.Node, group *v1.DiskGroup) error {
	dir := filepath.Join(ISCSIPath, node.Hostname)
	if err := btrfs.CreateSubvolume(dir); err != nil {
		return errors.Wrapf(err, "create lun subvolume %s", dir)
	}
	// the order of this list is also the lun ids
	for i, disk := range group.Disks {
		// assign the file path as the disk device
		disk.Device = filepath.Join(dir, fmt.Sprintf("%d.lun", i))
		disk.ID = int64(i + 1)
		if err := iscsi.NewLun(ctx, disk); err != nil {
			return errors.Wrapf(err, "create lun %d", i)
		}
	}
	return nil
}

func (c *Controller) installImage(ctx context.Context, group *v1.DiskGroup, i containerd.Image) error {
	var (
		disk      = group.Disks[0]
		diskMount = disk.Device + ".mnt"
		dest      = disk.Device + ".dest"
	)

	for _, p := range []string{diskMount, dest} {
		if err := os.MkdirAll(p, 0755); err != nil {
			return errors.Wrapf(err, "mkdir %s", p)
		}
	}
	if err := mountGroup(ctx, disk, diskMount); err != nil {
		return errors.Wrap(err, "mount group")
	}
	g, err := stage1.NewGroup(group, dest)
	if err != nil {
		return err
	}
	defer g.Close()

	if err := g.Init(diskMount); err != nil {
		return err
	}
	desc := i.Target()
	if err := image.Unpack(ctx, c.client.ContentStore(), &desc, dest); err != nil {
		return errors.Wrap(err, "unpack image to group")
	}

	if err := writeFstab(g, dest); err != nil {
		return errors.Wrap(err, "write fstab")
	}
	if err := c.writeResolvconf(dest); err != nil {
		return errors.Wrap(err, "write resolv.conf")
	}
	return nil
}

func writeFstab(g *stage1.Group, root string) error {
	entries := g.Entries()
	f, err := os.Create(filepath.Join(root, fstab.Path))
	if err != nil {
		return errors.Wrap(err, "create fstab file")
	}
	defer f.Close()
	if err := fstab.Write(f, entries); err != nil {
		return errors.Wrap(err, "write fstab")
	}
	return nil
}

func (c *Controller) writeResolvconf(root string) error {
	path := filepath.Join(root, resolvconf.DefaultPath)
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "create resolv.conf file %s", path)
	}
	defer f.Close()

	conf := &resolvconf.Conf{
		Nameservers: []string{
			c.ips[Gateway].To4().String(),
		},
	}
	return conf.Write(f)
}

func mountGroup(ctx context.Context, disk *v1.Disk, path string) error {
	out, err := exec.CommandContext(ctx, "mount", "-t", stage0.DefaultFilesystem, "-o", "loop", disk.Device, path).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func (c *Controller) createTarget(ctx context.Context, node *v1.Node) (target *v1.Target, err error) {
	iqn := iscsi.NewIQN(2024, "san.crosbymichael.com", node.Hostname, true)
	if node.InitiatorIqn == "" {
		return nil, errors.New("node does not have an initiator id")
	}
	for i := int64(1); i < MaxTargetID; i++ {
		if target, err = iscsi.NewTarget(ctx, iqn, i); err != nil {
			if !isTargetExists(err) {
				return nil, errors.Wrapf(err, "create target %s", iqn)
			}
			continue
		}
		break
	}
	if err := iscsi.Accept(ctx, target, node.InitiatorIqn); err != nil {
		return nil, errors.Wrap(err, "accept initiator iqn")
	}
	if err := iscsi.AcceptAllInitiators(ctx, target); err != nil {
		return nil, errors.Wrap(err, "accept ALL iqn")
	}
	return target, nil
}

func (c *Controller) writePXEConfig(node *v1.Node) error {
	p := &pxe.PXE{
		Default:      "terra",
		MAC:          node.Mac,
		InitiatorIQN: node.InitiatorIqn,
		// TODO: get target of the os disk group
		TargetIQN: node.DiskGroups[0].Target.Iqn,
		TargetIP:  c.ips[ISCSI].To4().String(),
		IP:        pxe.DHCP,
		Entries: []pxe.Entry{
			{
				Root:   "LABEL=os",
				Label:  "terra",
				Boot:   "terra",
				Kernel: c.kernel,
				Initrd: c.initrd,
				Append: []string{
					"version=os",
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

func withPlainRemote(ref string) containerd.RemoteOpt {
	return func(_ *containerd.Client, ctx *containerd.RemoteContext) error {
		var plain bool
		u, err := url.Parse("registry://" + ref)
		if err != nil {
			return errors.Wrap(err, "parse url")
		}
		plain = strings.Contains(u.Host, ":5000")
		ctx.Resolver = docker.NewResolver(docker.ResolverOptions{
			PlainHTTP: plain,
			Client:    http.DefaultClient,
		})
		return nil
	}
}
