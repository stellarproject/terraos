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

package agent

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/contrib/apparmor"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type stateChange interface {
	apply(context.Context) error
}

type stopDiff struct {
	a         *Agent
	container containerd.Container
}

func (s *stopDiff) apply(ctx context.Context) error {
	return s.a.stop(ctx, s.container)
}

type startDiff struct {
	a         *Agent
	container containerd.Container
}

func (s *startDiff) apply(ctx context.Context) error {
	return s.a.start(ctx, s.container)
}

func sameDiff() stateChange {
	return &nullDiff{}
}

type nullDiff struct {
}

func (n *nullDiff) apply(_ context.Context) error {
	return nil
}

func setupApparmor() error {
	return apparmor.WithDefaultProfile("orbit")(nil, nil, nil, &specs.Spec{
		Process: &specs.Process{},
	})
}
