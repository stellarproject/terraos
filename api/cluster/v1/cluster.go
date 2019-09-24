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

package v1

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrExists = errors.New("resource exists")
)

func (c *Cluster) RegisterMachine(ctx context.Context, machine *Machine) error {
	machine.UUID = uuid.New().String()
	c.Machines = append(c.Machines, machine)
	return nil
}

func (c *Cluster) RegisterVolume(ctx context.Context, v *Volume) error {
	for _, vv := range c.Volumes {
		if v.ID == vv.ID {
			return ErrExists
		}
	}
	c.Volumes = append(c.Volumes, v)
	return nil
}

func (c *Cluster) CreateNode(ctx context.Context, node *Node) error {
	for _, n := range c.Nodes {
		if n.Hostname == node.Hostname {
			return ErrExists
		}
	}
	c.Nodes = append(c.Nodes, node)
	return nil
}

func (n *Node) AttachVolume(ctx context.Context, v *Volume) error {
	for _, id := range n.VolumeIDs {
		if id == v.ID {
			return ErrExists
		}
	}
	n.VolumeIDs = append(n.VolumeIDs, v.ID)
	return nil
}

func (m *Machine) Deploy(ctx context.Context, n *Node) error {
	n.MachineID = m.UUID
	return nil
}
