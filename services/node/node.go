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

package node

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/node/v1"
)

var empty = &types.Empty{}

func New(n *v1.Node) (*Agent, error) {
	return &Agent{
		node: n,
		log:  logrus.WithField("hostname", n.Hostname),
	}, nil
}

type Agent struct {
	node *v1.Node
	log  logrus.Entry
}

func (a *Agent) Ping(ctx context.Context, _ *types.Empty) (*types.Empty, error) {
	a.log.Debug("node ping")
	return empty, nil
}

func (a *Agent) Info(ctx context.Context, _ *types.Empty) (*v1.InfoResponse, error) {
	return &v1.InfoResponse{
		Node: a.node,
	}, nil
}
