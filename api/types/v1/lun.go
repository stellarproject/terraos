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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrLunAttached = errors.New("lun attached to target")
)

const (
	AllIQN  = "ALL"
	OSLabel = "os"
)

func isLUNNotFound(err error) bool {
	return strings.Contains(err.Error(), "can't find the logical unit")
}

func NewTarget(ctx context.Context, id int64, iqn string) (*Target, error) {
	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "target",
		"--tid", strconv.Itoa(int(id)),
		"-T", iqn,
	); err != nil {
		return nil, errors.Wrapf(err, "%s", out)
	}
	return &Target{
		ID:  id,
		Iqn: iqn,
	}, nil
}

func (t *Target) Restore(ctx context.Context) error {
	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "target",
		"--tid", strconv.Itoa(int(t.ID)),
		"-T", t.Iqn,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	accepts := t.Accepts
	t.Accepts = nil
	for _, a := range accepts {
		if err := t.Accept(ctx, a); err != nil {
			return err
		}
	}
	for i := range t.Luns {
		if err := t.attachLocal(ctx, i+1); err != nil {
			return err
		}
	}
	return nil
}

func (t *Target) Delete(ctx context.Context) error {
	if len(t.Luns) > 0 {
		return ErrLunAttached
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

// Remove deletes the lun from the target
func (t *Target) Remove(ctx context.Context, l *LUN) error {
	index := t.getIndex(l)
	if index < 0 {
		return nil
	}
	if out, err := iscsi(ctx,
		"--op", "delete",
		"--mode", "logicalunit",
		"--tid", strconv.Itoa(int(t.ID)),
		"--lun", strconv.Itoa(int(index+1)),
	); err != nil {
		err = errors.Wrapf(err, "%s", out)
		if !isLUNNotFound(err) {
			return err
		}
	}
	// remote the lun by the index
	t.Luns = append(t.Luns[:index], t.Luns[index+1:]...)
	return nil
}

func (t *Target) getIndex(l *LUN) int {
	for i := range t.Luns {
		if t.Luns[i].ID == l.ID {
			return i
		}
	}
	return -1
}

func (t *Target) Attach(ctx context.Context, l *LUN) error {
	lid := len(t.Luns) + 1
	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "logicalunit",
		"--tid", strconv.Itoa(int(t.ID)),
		"--lun", strconv.Itoa(int(lid)),
		"-b", l.Path,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	t.Luns = append(t.Luns, l)
	return nil
}

func (t *Target) attachLocal(ctx context.Context, i int) error {
	l := t.Luns[i-1]
	if out, err := iscsi(ctx,
		"--op", "new",
		"--mode", "logicalunit",
		"--tid", strconv.Itoa(int(t.ID)),
		"--lun", strconv.Itoa(i),
		"-b", l.Path,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func (t *Target) Accept(ctx context.Context, iqn string) error {
	if out, err := iscsi(ctx,
		"--op", "bind",
		"--mode", "target",
		"--tid", strconv.Itoa(int(t.ID)),
		"-I", iqn,
	); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	t.Accepts = append(t.Accepts, iqn)
	return nil
}

// CreateLUN allocates a new lun with the specified size in MB
func CreateLUN(ctx context.Context, root, id string, size int64) (*LUN, error) {
	path := filepath.Join(root, id)
	args := []string{
		"--op", "new",
		"--device-type", "disk",
		"--type", "disk",
		"--size", strconv.Itoa(int(size)),
		"--file", path,
	}
	/* TODO: opts for thin
	if thin {
		args = append(args, "--thin-provisioning")
	}
	*/
	out, err := tgtimg(ctx, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", out)
	}
	return &LUN{
		ID:     id,
		FsSize: size,
		Path:   path,
	}, nil
}

func (l *LUN) Delete() error {
	return os.Remove(l.Path)
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
