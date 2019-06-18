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

package provision

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/node/v1"
	api "github.com/stellarproject/terraos/api/types/v1"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/stellarproject/terraos/remotes"
)

var empty = &types.Empty{}

type Service struct {
	client *containerd.Client
}

func (s *Service) Provision(ctx context.Context, r *v1.ProvisionRequest) (*types.Empty, error) {
	if r.Target == nil {
		return nil, errors.New("no target provided")
	}
	node := r.Node
	var (
		lun    *api.LUN
		volume *api.Volume
	)
	for _, l := range r.Target.Luns {
		if l.Label == api.OSLabel {
			lun = l
			break
		}
	}
	for _, v := range node.Volumes {
		if v.Label == api.OSLabel {
			volume = v
			break
		}
	}
	if lun == nil {
		return nil, errors.New("no os lun found")
	}
	if volume == nil {
		return nil, errors.New("no volume for lun found")
	}
	image, err := s.fetch(ctx, r.Image)
	if err != nil {
		return nil, err
	}
	if err := s.format(ctx, volume, lun); err != nil {
		return nil, err
	}
}

func (s *Service) fetch(ctx context.Context, repo string) (containerd.Image, error) {
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

func (s *Service) format(ctx context.Context, volume *api.Volume, l *api.LUN) error {
	args := []string{}
	if volume.FsType == "btrfs" {
		args = append(args, "-f")
	}
	args = append(args, l.Path)
	if err := mkfs.Mkfs(volume.FsType, l.Label, args...); err != nil {
		return errors.Wrapf(err, "format lun for %s with %s", l.Label, volume.FsType)
	}
	return nil
}
