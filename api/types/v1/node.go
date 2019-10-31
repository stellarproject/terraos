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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/iscsi"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"golang.org/x/sys/unix"
)

var (
	ErrNotISCSIVolume = errors.New("not an iscsi volume")
)

func (v *Volume) IsISCSI() bool {
	return v.TargetIqn != ""
}

func (v *Volume) Format(device string) error {
	return mkfs.Mkfs(v.Type, v.Label, device)
}

func (v *Volume) MountLabel() string {
	return fmt.Sprintf("LABEL=%s", v.Label)
}

func (v *Volume) Mount(device, dest string) (func() error, error) {
	p := filepath.Join(dest, v.Path)
	if err := os.MkdirAll(p, 0755); err != nil {
		return nil, errors.Wrapf(err, "mkdir %s", p)
	}
	if err := unix.Mount(device, p, v.Type, 0, ""); err != nil {
		return nil, errors.Wrapf(err, "mount %s to %s", v.Label, p)
	}
	return func() error {
		return unix.Unmount(p, 0)
	}, nil
}

func (v *Volume) Login(ctx context.Context, portal string) (string, error) {
	if !v.IsISCSI() {
		return "", ErrNotISCSIVolume
	}
	if err := iscsi.Discover(ctx, portal); err != nil {
		return "", err
	}
	target, err := iscsi.Login(ctx, portal, v.TargetIqn)
	if err != nil {
		return "", err
	}
	path := target.Partition(0, 1)
	if err := target.Ready(1 * time.Second); err != nil {
		target.Logout(ctx)
		return "", err
	}
	return path, nil
}

func (v *Volume) Logout(ctx context.Context, portal string) error {
	if !v.IsISCSI() {
		return ErrNotISCSIVolume
	}
	return iscsi.Logout(ctx, portal, v.TargetIqn)
}

func (v *Volume) Entries() []*fstab.Entry {
	return []*fstab.Entry{
		&fstab.Entry{
			Type:   v.Type,
			Pass:   2,
			Device: fmt.Sprintf("LABEL=%s", v.Label),
			Path:   v.Path,
		},
	}
}
