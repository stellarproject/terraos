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

package iscsi

import (
	"context"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	api "github.com/stellarproject/terraos/api/v1/infra"
	v1 "github.com/stellarproject/terraos/api/v1/types"
)

var empty = &types.Empty{}

type Controller struct {
	root string

	store *store
}

func isTargetExists(err error) bool {
	return strings.Contains(err.Error(), "this target already exists")
}

func New(root string, pool *redis.Pool) (*Controller, error) {
	// TODO: start tgt daemon
	c := &Controller{
		root: root,
		store: &store{
			pool: pool,
		},
	}
	ctx := context.Background()
	if err := c.restore(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Controller) restore(ctx context.Context) error {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	for _, t := range tran.State.Targets {
		if err := t.Restore(ctx); err != nil {
			return errors.Wrapf(err, "restore %s", t.Iqn)
		}
	}
	return nil
}

func (c *Controller) CreateTarget(ctx context.Context, r *api.CreateTargetRequest) (*api.CreateTargetResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	i := len(tran.State.Targets)
	target, err := v1.NewTarget(ctx, int64(i+1), r.Iqn)
	if err != nil {
		return nil, err
	}
	tran.State.Targets = append(tran.State.Targets, target)
	if err := tran.Commit(ctx); err != nil {
		return nil, err
	}
	return &api.CreateTargetResponse{
		Target: target,
	}, nil
}

func (c *Controller) CreateLUN(ctx context.Context, r *api.CreateLUNRequest) (*api.CreateLUNResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	if _, ok := tran.State.UnallocatedLuns[r.ID]; ok {
		return nil, ErrLunExists
	}
	lun, err := v1.CreateLUN(ctx, c.root, r.ID, r.FsSize)
	if err != nil {
		return nil, err
	}
	if tran.State.UnallocatedLuns == nil {
		tran.State.UnallocatedLuns = make(map[string]*v1.LUN)
	}
	tran.State.UnallocatedLuns[r.ID] = lun
	if err := tran.Commit(ctx); err != nil {
		return nil, err
	}
	return &api.CreateLUNResponse{
		Lun: lun,
	}, nil
}

func (c *Controller) AttachLUN(ctx context.Context, r *api.AttachLUNRequest) (*api.AttachLUNResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	var target *v1.Target
	for _, t := range tran.State.Targets {
		if t.Iqn == r.Target.Iqn {
			target = t
			break
		}
	}
	if target == nil {
		return nil, ErrTargetNotExist
	}
	if err := target.Attach(ctx, r.Lun); err != nil {
		return nil, err
	}
	delete(tran.State.UnallocatedLuns, r.Lun.ID)
	if err := tran.Commit(ctx); err != nil {
		return nil, err
	}
	return &api.AttachLUNResponse{
		Target: target,
	}, nil
}

func (c *Controller) ListTargets(ctx context.Context, r *types.Empty) (*api.ListTargetsResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()
	return &api.ListTargetsResponse{
		Targets: tran.State.Targets,
	}, nil
}

func (c *Controller) DeleteTarget(ctx context.Context, r *api.DeleteTargetRequest) (*types.Empty, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	var (
		index  int
		target *v1.Target
	)
	for i, t := range tran.State.Targets {
		if t.Iqn == r.Iqn {
			target = t
			index = i
			break
		}
	}
	if target == nil {
		return nil, ErrTargetNotExist
	}
	if err := target.Delete(ctx); err != nil {
		return nil, err
	}
	tran.State.Targets = append(tran.State.Targets[:index], tran.State.Targets[index+1:]...)
	if err := tran.Commit(ctx); err != nil {
		return nil, err
	}
	return empty, nil
}

func (c *Controller) DeleteLUN(ctx context.Context, r *api.DeleteLUNRequest) (*types.Empty, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	if lun, ok := tran.State.UnallocatedLuns[r.ID]; ok {
		if err := lun.Delete(); err != nil {
			return nil, err
		}
		delete(tran.State.UnallocatedLuns, r.ID)
		return empty, tran.Commit(ctx)
	}
	for _, t := range tran.State.Targets {
		for _, lun := range t.Luns {
			if lun.ID == r.ID {
				if err := t.Remove(ctx, lun); err != nil {
					return nil, err
				}
				if err := lun.Delete(); err != nil {
					return nil, err
				}
				return empty, tran.Commit(ctx)
			}
		}
	}
	return nil, ErrLUNNotExist
}
