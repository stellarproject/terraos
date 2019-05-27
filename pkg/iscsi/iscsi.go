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
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/api/v1/types"
)

const (
	targetIqnFmt = "iqn.%d.%s.%s:fs"
	nodeIqnFmt   = "iqn.%d.%s:%s"
)

func Check() error {
	if _, err := exec.LookPath("tgtadm"); err != nil {
		return errors.Wrap(err, "tgtadm command cannot be found")
	}
	return nil
}

func NewIQN(year int, domain, machine string, fs bool) string {
	if fs {
		return fmt.Sprintf(targetIqnFmt, year, domain, machine)
	}
	return fmt.Sprintf(nodeIqnFmt, year, domain, machine)
}

func iscsi(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "tgtadm", append([]string{
		"--lld", "iscsi",
	}, args...)...)
	return cmd.CombinedOutput()
}

func tgtimg(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "tgtimg", args...).CombinedOutput()
}

func NewTarget(ctx context.Context, iqn string, tid int64) (*types.Target, error) {
	t := &types.Target{
		Iqn: string(iqn),
		ID:  tid,
	}
	return t, SetTarget(ctx, t)
}

func SetTarget(ctx context.Context, t *types.Target) error {
	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "target",
		"--tid", strconv.Itoa(int(t.ID)),
		"-T", t.Iqn,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func AcceptAllInitiators(ctx context.Context, t *types.Target) error {
	if out, err := iscsi(ctx,
		"--op", "bind",
		"--mode", "target",
		"--tid", strconv.Itoa(int(t.ID)),
		"-I", "ALL",
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func Accept(ctx context.Context, t *types.Target, iqn string) error {
	if out, err := iscsi(ctx,
		"--op", "bind",
		"--mode", "target",
		"--tid", strconv.Itoa(int(t.ID)),
		"-I", iqn,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

// Attach a lun to the target
func Attach(ctx context.Context, t *types.Target, l *types.Disk) error {
	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "logicalunit",
		"--tid", strconv.Itoa(int(t.ID)),
		"--lun", strconv.Itoa(int(l.ID)),
		"-b", l.Device,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func Delete(ctx context.Context, t *types.Target, lun *types.Disk) error {
	if lun != nil {
		if out, err := iscsi(ctx,
			"--op", "delete",
			"--mode", "logicalunit",
			"--tid", strconv.Itoa(int(t.ID)),
			"--lun", strconv.Itoa(int(lun.ID)),
		); err != nil {
			return errors.Wrapf(err, "%s", out)
		}
	}
	if out, err := iscsi(ctx,
		"--op", "delete",
		"--mode", "target",
		"--tid", strconv.Itoa(int(t.ID)),
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

// NewLun allocates a new lun with the specified size in MB
func NewLun(ctx context.Context, id int64, disk *types.Disk) error {
	args := []string{
		"--op", "new",
		"--device-type", "disk",
		"--size", strconv.Itoa(int(disk.FsSize)),
		"--file", disk.Device,
	}
	/* TODO: opts for thin
	if thin {
		args = append(args, "--thin-provisioning")
	}
	*/
	out, err := tgtimg(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func DeleteLun(l *types.Disk) error {
	return os.Remove(l.Device)
}
