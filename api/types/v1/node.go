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
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"golang.org/x/sys/unix"
)

const (
	year       = 2019
	nodeIqnFmt = "iqn.%d.%s:%s"
	initIQN    = "iqn.%d.%s:%s.%s"
)

func (n *Node) InitiatorIQN() string {
	return fmt.Sprintf(initIQN, year, n.Domain, "node", n.Hostname)
}

func (n *Node) IQN() string {
	return fmt.Sprintf(nodeIqnFmt, year, n.Domain, n.Hostname)
}

func (v *Volume) Format(device string) error {
	return mkfs.Mkfs(v.Type, v.Label, device)
}

func (v *Volume) Mount(device, path string) (func() error, error) {
	if err := unix.Mount(device, filepath.Join(path, v.Path), v.Type, 0, ""); err != nil {
		return nil, errors.Wrapf(err, "mount %s to %s", v.Label, path)
	}
	return func() error {
		return unix.Unmount(path, 0)
	}, nil
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
