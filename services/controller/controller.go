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
	"os"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd/namespaces"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	api "github.com/stellarproject/terraos/api/controller/v1"
	iscsi "github.com/stellarproject/terraos/api/iscsi/v1"
	v1 "github.com/stellarproject/terraos/api/node/v1"
	pxe "github.com/stellarproject/terraos/api/pxe/v1"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/remotes"
)

const KeyNodes = "io.stellarproject.nodes"

func New() (*Controller, error) {
	if err := btrfs.Check(); err != nil {
		return nil, err
	}
	// TODO: move to redis
	if err := remotes.LoadRemotes(remotes.DefaultPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	c := &Controller{}
	return c, nil
}

type Controller struct {
	mu sync.Mutex

	pool *redis.Pool

	iscsi       iscsi.ServiceClient
	provisioner v1.ProvisionerClient
	pxe         pxe.ServiceClient
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
	return nil
}

func (c *Controller) Register(ctx context.Context, r *api.RegisterNodeRequest) (_ *api.RegisterNodeResponse, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx = namespaces.WithNamespace(ctx, "controller")

	node := r.Node
	// TODO: validate node
	// validate nics has > 0

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
	targets, err := c.createVolumes(ctx, node)
	if err != nil {
		return errors.Wrap(err, "create volumes")
	}
	var pxeIscsi *pxe.ISCSI
	if t, ok := targets["os"]; ok {
		pxeIscsi = &pxe.ISCSI{
			InitiatorIqn: node.IQN(),
			TargetIqn:    t.Iqn,
			TargetIp:     c.iscsiIP,
		}
	}
	if _, err := c.provisioner.Provision(ctx, &v1.ProvisionRequest{
		Image: c.image,
		Node:  node,
	}); err != nil {
		return nil, errors.Wrap(err, "provision node")
	}
	// TODO: fix mac registration
	nic := node.Nics[0]
	if _, err := c.pxe.Register(ctx, &pxe.RegisterRequest{
		Mac:   nic.Mac,
		Ip:    nic.Ip,
		Root:  "LABEL=os",
		Boot:  "pxe",
		Iscsi: pxeIscsi,
		// TODO: fix os_volume mount in initrd
	}); err != nil {
		return nil, errors.Wrap(err, "register node with pxe")
	}
	if err := c.provisionTarget(ctx, node, image); err != nil {
		return nil, errors.Wrap(err, "provision node target")
	}
	return &api.ProvisionNodeResponse{
		Node: node,
	}, nil
}

func (c *Controller) createVolumes(ctx context.Context, node *v1.Node) (*iscsi.Target, error) {
	targetResponse, err := c.iscsi.CreateTarget(ctx, &iscsi.CreateTargetRequest{
		Iqn: v.IQN(node),
	})
	if err != nil {
		return nil, errors.Wrap(err, "create target")
	}
	target := targetResponse.Target
	for _, v := range node.Volumes {
		if v.Type != v1.ISCSIVolume {
			continue
		}
		lunResponse, err := c.iscsi.CreateLUN(ctx, &iscsi.CreateLUNRequest{
			ID:     fmt.Sprintf("%s.%s", node.Hostname, v.Label),
			FsSize: v.FsSize,
		})
		if err != nil {
			return nil, errors.Wrap(err, "create lun")
		}
		final, err := c.iscsi.AttachLUN(ctx, &iscsi.AttachLUNRequest{
			Target: target,
			Lun:    lunResponse.Lun,
		})
		if err != nil {
			return nil, errors.Wrap(err, "attach lun to target")
		}
		target = final.Target
	}
	return target, nil
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
