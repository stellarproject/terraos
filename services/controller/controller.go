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
	"sync"

	"github.com/containerd/containerd/namespaces"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	api "github.com/stellarproject/terraos/api/controller/v1"
	pxe "github.com/stellarproject/terraos/api/pxe/v1"
	v1 "github.com/stellarproject/terraos/api/types/v1"
	"github.com/stellarproject/terraos/util"
	"google.golang.org/grpc"
)

const (
	KeyNodes       = "io.stellarproject.nodes"
	KeyISCSIServer = "io.stellarproject.iscsi/address"
)

func New(pool *redis.Pool) (*Controller, error) {
	ip, gateway, err := util.IPAndGateway()
	if err != nil {
		return nil, err
	}
	c := &Controller{
		pool: pool,
	}
	return c, nil
}

type Controller struct {
	mu sync.Mutex

	pool *redis.Pool
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

	if _, err := conn.Do("HDEL", KeyNodes, node.Hostname); err != nil {
		return errors.Wrap(err, "delete node from kv")
	}
	return nil
}

func (c *Controller) Register(ctx context.Context, r *api.RegisterNodeRequest) (_ *api.RegisterNodeResponse, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node := r.Node
	if len(node.Nics) == 0 {
		return nil, errors.New("node NICs required for registration")
	}

	ctx = namespaces.WithNamespace(ctx, "controller")
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	addr, err := redis.String(conn.Do("GET", KeyISCSIServer))
	if err != nil {
		return nil, errors.Wrap(err, "get iscsi address")
	}

	// do the initial save so we know this host does not exist
	if err := c.saveNode(node); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if err := c.delete(ctx, node); err != nil {
				logrus.WithError(err).Error("delete failed node")
			}
		}
	}()
	var pxeIscsi *pxe.ISCSI
	for _, l := range node.Volumes {
		if l.Label == v1.OSLabel {
			pxeIscsi = &pxe.ISCSI{
				InitiatorIqn: node.InitiatorIQN(),
				TargetIqn:    l.TargetIqn,
				TargetIp:     addr,
			}
			break
		}
	}

	gconn, err := grpc.Dial("127.0.0.1:9000", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer gconn.Close()
	pxeClient := pxe.NewServiceClient(gconn)

	// TODO: fix mac registration when more than one nics
	var (
		ip  string
		nic = node.Nics[0]
	)
	if len(nic.Addresses) == 0 {
		ip = "dhcp"
	} else {
		ip = nic.Addresses[0]
	}
	if _, err := pxeClient.Register(ctx, &pxe.RegisterRequest{
		Mac:   nic.Mac,
		Ip:    ip,
		Root:  "LABEL=os",
		Boot:  "pxe",
		Iscsi: pxeIscsi,
		// TODO: fix os_volume mount in initrd
	}); err != nil {
		return nil, errors.Wrap(err, "register node with pxe")
	}
	return &api.RegisterNodeResponse{
		Node:   node,
		Target: target,
	}, nil
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
