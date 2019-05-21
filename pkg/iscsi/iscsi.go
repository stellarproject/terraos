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
	"os/exec"
	"strconv"
	"sync"

	"github.com/pkg/errors"
)

const (
	targetIqnFmt = "iqn.%d.%s.%s:%d"
	nodeIqnFmt   = "iqn.%d.%s:%s"
)

func Check() error {
	if _, err := exec.LookPath("tgtadm"); err != nil {
		return errors.Wrap(err, "tgtadm command cannot be found")
	}
	return nil
}

type IQN string

func NewIQN(year int, domain, machine string, disk int) IQN {
	if disk > -1 {
		return IQN(fmt.Sprintf(targetIqnFmt, year, domain, machine, disk))
	}
	return IQN(fmt.Sprintf(nodeIqnFmt, year, domain, machine))
}

func iscsi(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "tgtadm", append([]string{
		"--lld", "iscsi",
	}, args...)...)
	return cmd.CombinedOutput()
}

func NewTarget(ctx context.Context, iqn IQN, tid int) (*Target, error) {
	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "target",
		"--tid", strconv.Itoa(tid),
		"-T", string(iqn),
	); err != nil {
		return nil, errors.Wrapf(err, "%s", out)
	}
	return &Target{
		iqn: iqn,
		tid: tid,
	}, nil
}

type Target struct {
	mu sync.Mutex

	iqn IQN
	tid int
}

func (t *Target) IQN() IQN {
	return t.iqn
}

func (t *Target) ID() int {
	return t.tid
}

func (t *Target) AcceptAllInitiators(ctx context.Context) error {
	if out, err := iscsi(ctx,
		"--op", "bind",
		"--mode", "target",
		"--tid", strconv.Itoa(t.tid),
		"-I", "ALL",
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

// Attach a lun to the target
func (t *Target) Attach(ctx context.Context, l *Lun, lunID int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "logicalunit",
		"--tid", strconv.Itoa(t.tid),
		"--lun", strconv.Itoa(lunID),
		"-b", l.path,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	l.lid = lunID
	return nil
}

// NewLun allocates a new lun with the specified size in MB
func NewLun(ctx context.Context, path string, size int64) (*Lun, error) {
	out, err := exec.CommandContext(ctx, "dd",
		"if=/dev/zero",
		"of="+path,
		"bs=1M",
		fmt.Sprintf("count=%d", size),
	).CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "%s", out)
	}
	return &Lun{
		path: path,
		size: size,
	}, nil
}

type Lun struct {
	lid  int
	path string
	size int64
}

func (l *Lun) ID() int {
	return l.lid
}

func (l *Lun) Path() string {
	return l.path
}

// Size in MB
func (l *Lun) Size() int64 {
	return l.size
}

func (l *Lun) LocalMount(ctx context.Context, t, path string) error {
	// TODO: make into syscall mount command
	out, err := exec.CommandContext(ctx, "mount", "-t", t, "-o", "loop", l.path, path).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
