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
	"fmt"
	"os"
	"os/exec"

	"github.com/containerd/containerd"
	"github.com/nats-io/go-nats/encoders/protobuf"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/api/v1/messages"
)

type Node struct {
	hostname string
	nc       *nats.Conn
	conn     *nats.EncodedConn
	errCh    chan error
	log      *logrus.Entry
	subs     []*nats.Subscription
	client   *containerd.Client
}

func New(addr string) (*Node, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "get hostname")
	}
	nc, err := nats.Connect(addr)
	if err != nil {
		return nil, errors.Wrap(err, "connect to nats")
	}
	conn, err := nats.NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return nil, errors.Wrap(err, "protobuf nats encoder")
	}
	return &Node{
		hostname: hostname,
		nc:       nc,
		conn:     conn,
		errCh:    make(chan error, 128),
		log:      logrus.WithField("hostname", hostname),
	}, nil
}

func (n *Node) Start(ctx context.Context) (<-chan error, error) {
	s, err := n.conn.Subscribe(n.nodeSubject(messages.NodeUpdateSubject), n.terraUpdate)
	if err != nil {
		return nil, errors.Wrap(err, "subscribe to updates")
	}
	n.subs = append(n.subs, s)
	if s, err := n.conn.Subscribe(n.nodeSubject(messages.NodeShutdownSubject), n.shutdown); err != nil {
		return nil, errors.Wrap(err, "subscribe to shutdown")
	}
	n.subs = append(n.subs, s)

	// start the ping loop
	go n.pingLoop(ctx)

	return n.errCh, nil
}

// update updates a nodes binary, not the os volume
func (n *Node) terraUpdate(r *messages.NodeUpdate) {
	panic("not implemented")
}

func (n *Node) nodeSubject(tm string) string {
	return fmt.Sprintf(tm, n.hostname)
}

func (n *Node) shutdown(r *messages.Shutdown) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var args []string
	if r.Restart {
		args = append(args, "--reboot")
	}
	if r.Now {
		args = append(args, "--halt")
	}
	cmd := exec.CommandContext(ctx, "shutdown", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		n.log.WithError(err).Error("run shutdown command")
		n.errCh <- err
	}
}

func (n *Node) pingLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n.log.Debug("publish ping")
			m := &messages.Ping{Hostname: n.hostname}
			if err := n.conn.Publish(messages.PingSubject, m); err != nil {
				n.log.WithError(err).Error("publish ping")
				n.errCh <- err
			}
		}
	}
}

func (n *Node) Close() error {
	for _, s := range n.subs {
		if err := s.Unsubscribe(); err != nil {
			n.log.WithError(err).Error("unsubscribe")
		}
	}
	n.nc.Close()
	return nil
}
