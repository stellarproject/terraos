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

package stage0

import (
	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/api/v1/types"
	"github.com/stellarproject/terraos/pkg/grub"
	"golang.org/x/sys/unix"
)

const (
	Image      = "docker.io/stellarproject/pxe"
	Stage0Path = "/boot"
)

func Overlay(group *types.DiskGroup) (func() error, error) {
	if err := unix.Mount(group.Disks[0].Device, Stage0Path, stage0Filesystem, 0, ""); err != nil {
		return nil, errors.Wrap(err, "overlay stage0")
	}
	return func() error {
		return unix.Unmount(Stage0Path, 0)
	}, nil
}

func MBR(device string) error {
	if err := grub.MkConfig(Stage0Path); err != nil {
		return errors.Wrap(err, "make grub config")
	}
	if err := grub.Install(device); err != nil {
		return errors.Wrapf(err, "install grub to %s", device)
	}
	return nil
}
