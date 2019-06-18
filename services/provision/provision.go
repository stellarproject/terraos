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
	"bytes"
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/node/v1"
	api "github.com/stellarproject/terraos/api/types/v1"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/stellarproject/terraos/remotes"
	"golang.org/x/sys/unix"
)

var empty = &types.Empty{}

type Service struct {
	// TODO: can't run this in a container because of this dependency
	client *containerd.Client
}

func (s *Service) Provision(ctx context.Context, r *v1.ProvisionRequest) (*types.Empty, error) {
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
			return errors.Errorf("volume does not exist for %s", label)
		}
		if err := s.format(ctx, volume, lun); err != nil {
			return nil, err
		}
	}
	// not all nodes will have a lun for the OS install
	if r.Image != "" {
		lun, ok := luns[types.OSLabel]
		if !ok {
			return errors.New("no lun for os install")
		}
		volume, ok := volumes[types.OSLabel]
		if !ok {
			return errors.New("no os volume for os install")
		}
		if volume.FsType != "ext4" {
			return errors.Errorf("only ext4 is supported for luns: %s", volume.FsType)
		}
		image, err := s.fetch(ctx, r.Image)
		if err != nil {
			return nil, err
		}
		if err := s.installImage(ctx, lun, image, r); err != nil {
			return nil, err
		}
	}
	return empty, nil
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
	if err := mkfs.Mkfs(volume.FsType, l.Label, l.Path); err != nil {
		return errors.Wrapf(err, "format lun for %s with %s", l.Label, volume.FsType)
	}
	return nil
}

func (s *Service) installImage(ctx context.Context, volume *api.Volume, l *api.LUN, i containerd.Image, r *v1.ProvisionRequest) error {
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
}

func (s *Service) applyConfigurationLayer(ctx context.Context, r *v1.ProvisionRequest, dest string) error {
	cmd := exec.CommandContext(ctx, "terra", "_configure")

	data, err := proto.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request")
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
