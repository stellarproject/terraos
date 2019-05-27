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
	"github.com/stellarproject/terraos/pkg/mkfs"
)

const (
	DefaultFilesystem = "btrfs"
	stage0Filesystem  = "fat32"
)

func Format(g *types.DiskGroup) error {
	var (
		devices []string
		args    = []string{"-f"}
		t       = DefaultFilesystem
	)
	if g.Stage == types.Stage0 {
		args = []string{}
		t = stage0Filesystem
	}
	switch g.GroupType {
	case types.Single:
		if len(g.Disks) != 1 {
			return errors.New("cannot have a single group with multiple disks")
		}
		devices = append(devices, g.Disks[0].Device)
	case types.RAID0:
		args = append(args, "-d", "raid0")
		for _, d := range g.Disks {
			devices = append(devices, d.Device)
		}
	case types.RAID5:
		args = append(args, "-d", "raid5", "-m", "raid5")
		for _, d := range g.Disks {
			devices = append(devices, d.Device)
		}
	case types.RAID10:
		args = append(args, "-d", "raid10", "-m", "raid10")
		for _, d := range g.Disks {
			devices = append(devices, d.Device)
		}
	default:
		return errors.Errorf("unsupported group type %s", g.GroupType)
	}
	if err := mkfs.Mkfs(DefaultFilesystem, g.Label, append(args, devices...)...); err != nil {
		return errors.Wrapf(err, "mkfs of disk group for %s", g.Label)
	}
	return nil
}
