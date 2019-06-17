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

package pxe

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/content"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/pxe/v1"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/pxe"
	pp "github.com/stellarproject/terraos/pkg/pxe"
	"github.com/stellarproject/terraos/remotes"
)

var empty = &types.Empty{}

const (
	PXEKey     = "io.stellarproject.pxe/bootloaders"
	DefaultKey = "default"
	kernel     = "vmlinuz"
	initrd     = "initrd.img"
	configDir  = "pxelinux.cfg"
)

func New(path string, pool *redis.Pool, store content.Store) (*Service, error) {
	return &Service{
		pool:  pool,
		path:  path,
		store: store,
	}, nil
}

type Service struct {
	path  string
	pool  *redis.Pool
	store content.Store
}

func (s *Service) Install(ctx context.Context, r *v1.InstallRequest) (*types.Empty, error) {
	b := r.Loader
	if b.ID == DefaultKey {
		return nil, errors.New("invalid reserved id")
	}
	data, err := proto.Marshal(b)
	if err != nil {
		return nil, errors.Wrap(err, "marshal proto for save")
	}
	conn, err := s.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()

	if _, err := conn.Do("HSETNX", PXEKey, b.ID, data); err != nil {
		return nil, errors.Wrap(err, "set pxe version")
	}
	i, err := image.Fetch(ctx, remotes.Plain(b.Image), s.store, b.Image)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch pxe image %s", b.Image)
	}
	path, err := ioutil.TempDir("", "terra-pxe-install")
	if err != nil {
		return nil, errors.Wrap(err, "create tmp pxe dir")
	}
	defer os.RemoveAll(path)

	if err := image.Unpack(ctx, s.store, i, path); err != nil {
		return nil, errors.Wrap(err, "unpack pxe image")
	}
	if err := syncDir(ctx, b.ID, filepath.Join(path, "tftp")+"/", s.path+"/"); err != nil {
		return nil, errors.Wrap(err, "sync tftp dir")
	}
	if r.Default {
		if _, err := conn.Do("HSET", PXEKey, DefaultKey, data); err != nil {
			return nil, errors.Wrap(err, "set bootloader as default")
		}
	}
	return empty, nil
}

func (s *Service) List(ctx context.Context, _ *types.Empty) (*v1.ListResponse, error) {
	conn, err := s.pool.GetContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get connection")
	}
	defer conn.Close()
	data, err := redis.StringMap(conn.Do("HGETALL", PXEKey))
	if err != nil {
		return nil, errors.Wrap(err, "get all bootloaders")
	}
	var resp v1.ListResponse
	for k, v := range data {
		var b v1.Bootloader
		if err := proto.Unmarshal([]byte(v), &b); err != nil {
			return nil, errors.Wrapf(err, "unmarshal %s", k)
		}
		resp.Bootloaders = append(resp.Bootloaders, &b)
	}
	return &resp, nil
}

func (s *Service) Register(ctx context.Context, r *v1.RegisterRequest) (*types.Empty, error) {
	ip := r.Ip
	if ip == "" {
		ip = pp.DHCP
	}
	if r.Loader == nil {
		l, err := s.getDefault(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "get default bootloader")
		}
		r.Loader = l
	}
	id := r.Loader.ID
	p := &pp.PXE{
		Default: "pxe",
		MAC:     r.Mac,
		IP:      ip,
		Entries: []pxe.Entry{
			{
				Root:   "LABEL=os",
				Label:  "pxe",
				Boot:   r.Boot,
				Kernel: filepath.Join("/", id, kernel),
				Initrd: filepath.Join("/", id, initrd),
				Append: r.Options,
			},
		},
	}
	if i := r.Iscsi; i != nil {
		p.TargetIP = i.TargetIp
		p.TargetIQN = i.TargetIqn
		p.InitiatorIQN = i.InitiatorIqn
	}
	path := filepath.Join(s.path, configDir, p.Filename())
	f, err := os.Create(path)
	if err != nil {
		return nil, errors.Wrapf(err, "create pxe file %s", path)
	}
	defer f.Close()

	if err := p.Write(f); err != nil {
		return nil, errors.Wrap(err, "write pxe config")
	}
	return empty, nil
}

func (s *Service) getDefault(ctx context.Context) (*v1.Bootloader, error) {
	conn, err := s.pool.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("HGET", PXEKey, DefaultKey))
	if err != nil {
		return nil, err
	}

	var b v1.Bootloader
	if err := proto.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Service) Remove(ctx context.Context, r *v1.RemoveRequest) (*types.Empty, error) {
	p := pp.PXE{
		MAC: r.Mac,
	}
	path := filepath.Join(s.path, configDir, p.Filename())
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "delete pxe config %s", path)
		}
	}
	return empty, nil
}

func syncDir(ctx context.Context, id, source, target string) error {
	path := filepath.Join(target, id)
	if err := os.MkdirAll(path, 0711); err != nil {
		return errors.Wrapf(err, "mkdir %s", path)
	}
	for _, f := range []string{
		filepath.Join(source, kernel),
		filepath.Join(source, initrd),
	} {
		to := filepath.Join(path, filepath.Base(f))
		if err := os.Rename(f, to); err != nil {
			return errors.Wrapf(err, "rename %s to %s", f, to)
		}
	}
	return nil
}
