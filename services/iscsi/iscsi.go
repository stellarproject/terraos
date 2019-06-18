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
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/docker/docker/errdefs"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/iscsi/v1"
	prov "github.com/stellarproject/terraos/api/node/v1"
	api "github.com/stellarproject/terraos/api/types/v1"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/stellarproject/terraos/remotes"
	"golang.org/x/sys/unix"
)

var empty = &types.Empty{}

type Controller struct {
	root   string
	client *containerd.Client

	store *store
}

func isTargetExists(err error) bool {
	return strings.Contains(err.Error(), "this target already exists")
}

func New(pool *redis.Pool, client *containerd.Client) (*Controller, error) {
	c := &Controller{
		root:   "/iscsi",
		client: client,
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
		return err
	}
	defer tran.Rollback()

	for _, t := range tran.State.Targets {
		if err := t.Restore(ctx); err != nil {
			return errors.Wrapf(err, "restore %s", t.Iqn)
		}
	}
	return nil
}

func (c *Controller) CreateTarget(ctx context.Context, r *v1.CreateTargetRequest) (*v1.CreateTargetResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	i := len(tran.State.Targets)
	target, err := api.NewTarget(ctx, int64(i+1), r.Iqn)
	if err != nil {
		return nil, err
	}
	tran.State.Targets = append(tran.State.Targets, target)
	if err := tran.Commit(ctx); err != nil {
		return nil, err
	}
	return &v1.CreateTargetResponse{
		Target: target,
	}, nil
}

func (c *Controller) CreateLUN(ctx context.Context, r *v1.CreateLUNRequest) (*v1.CreateLUNResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	if _, ok := tran.State.UnallocatedLuns[r.ID]; ok {
		return nil, ErrLunExists
	}
	lun, err := api.CreateLUN(ctx, c.root, r.ID, r.FsSize)
	if err != nil {
		return nil, err
	}
	if tran.State.UnallocatedLuns == nil {
		tran.State.UnallocatedLuns = make(map[string]*api.LUN)
	}
	tran.State.UnallocatedLuns[r.ID] = lun
	if err := tran.Commit(ctx); err != nil {
		return nil, err
	}
	return &v1.CreateLUNResponse{
		Lun: lun,
	}, nil
}

func (c *Controller) AttachLUN(ctx context.Context, r *v1.AttachLUNRequest) (*v1.AttachLUNResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	var target *api.Target
	for _, t := range tran.State.Targets {
		if t.Iqn == r.TargetIqn {
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
	return &v1.AttachLUNResponse{
		Target: target,
	}, nil
}

func (c *Controller) ListTargets(ctx context.Context, r *types.Empty) (*v1.ListTargetsResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()
	return &v1.ListTargetsResponse{
		Targets: tran.State.Targets,
	}, nil
}

func (c *Controller) Get(ctx context.Context, r *v1.GetRequest) (*v1.GetResponse, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()
	var resp v1.GetResponse
	for _, t := range tran.State.Targets {
		if t.Iqn == r.Iqn {
			resp.Target = t
			break
		}
	}
	if resp.Target == nil {
		return nil, errors.Wrap(os.ErrNotExist, "target does not exist")
	}
	return &resp, nil
}

func (c *Controller) DeleteTarget(ctx context.Context, r *v1.DeleteTargetRequest) (*types.Empty, error) {
	tran, err := c.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tran.Rollback()

	var (
		index  int
		target *api.Target
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

func (c *Controller) DeleteLUN(ctx context.Context, r *v1.DeleteLUNRequest) (*types.Empty, error) {
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

func (s *Controller) Provision(ctx context.Context, r *prov.ProvisionRequest) (*types.Empty, error) {
	if r.Target == nil {
		return nil, errors.New("no target provided")
	}
	node := r.Node
	var (
		luns    = make(map[string]*api.LUN)
		volumes = make(map[string]*api.Volume)
	)
	for _, l := range r.Target.Luns {
		luns[l.Label] = l
	}
	for _, v := range node.Volumes {
		volumes[v.Label] = v
	}
	// format all the luns for the target
	for label, lun := range luns {
		volume, ok := volumes[label]
		if !ok {
			return nil, errors.Errorf("volume does not exist for %s", label)
		}
		if err := s.format(ctx, volume, lun); err != nil {
			return nil, err
		}
	}
	// not all nodes will have a lun for the OS install
	if r.Image != "" {
		lun, ok := luns[api.OSLabel]
		if !ok {
			return nil, errors.New("no lun for os install")
		}
		volume, ok := volumes[api.OSLabel]
		if !ok {
			return nil, errors.New("no os volume for os install")
		}
		if volume.FsType != "ext4" {
			return nil, errors.Errorf("only ext4 is supported for luns: %s", volume.FsType)
		}
		image, err := s.fetch(ctx, r.Image)
		if err != nil {
			return nil, err
		}
		if err := s.installImage(ctx, volume, lun, image, r); err != nil {
			return nil, err
		}
	}
	return empty, nil
}

func (s *Controller) fetch(ctx context.Context, repo string) (containerd.Image, error) {
	image, err := s.client.GetImage(ctx, repo)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, errors.Wrapf(err, "image get error %s", repo)
		}
		i, err := s.client.Fetch(ctx, repo, remotes.WithPlainRemote(repo))
		if err != nil {
			return nil, errors.Wrapf(err, "fetch repo %s", repo)
		}
		image = containerd.NewImage(s.client, i)
	}
	return image, nil
}

func (s *Controller) format(ctx context.Context, volume *api.Volume, l *api.LUN) error {
	if err := mkfs.Mkfs(volume.FsType, l.Label, l.Path); err != nil {
		return errors.Wrapf(err, "format lun for %s with %s", l.Label, volume.FsType)
	}
	return nil
}

func (s *Controller) installImage(ctx context.Context, volume *api.Volume, l *api.LUN, i containerd.Image, r *prov.ProvisionRequest) error {
	dest := l.Path + ".dest"
	if err := os.MkdirAll(dest, 0755); err != nil {
		return errors.Wrapf(err, "mkdir lun dest %s", dest)
	}
	defer os.Remove(dest)

	if err := mount(ctx, volume.FsType, l.Path, dest); err != nil {
		return errors.Wrap(err, "mount lun for install")
	}
	defer unix.Unmount(dest, 0)

	desc := i.Target()
	if err := image.Unpack(ctx, s.client.ContentStore(), &desc, dest); err != nil {
		return errors.Wrap(err, "unpack image to lun")
	}
	// apply configuration
	if err := s.applyConfigurationLayer(ctx, r, dest); err != nil {
		return errors.Wrap(err, "apply configuration")
	}
	return nil
}

func (s *Controller) applyConfigurationLayer(ctx context.Context, r *prov.ProvisionRequest, dest string) error {
	cmd := exec.CommandContext(ctx, "terra-configure")

	data, err := proto.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "marshal request")
	}
	buf := bytes.NewReader(data)
	cmd.Stdin = buf
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Chroot = dest
	cmd.Dir = "/"
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func mount(ctx context.Context, fstype, lunPath, dest string) error {
	out, err := exec.CommandContext(ctx, "mount", "-t", fstype, "-o", "loop", lunPath, dest).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
