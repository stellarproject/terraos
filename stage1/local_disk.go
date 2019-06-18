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

package stage1

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	types "github.com/stellarproject/terraos/api/node/v1"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"golang.org/x/sys/unix"
)

const (
	OSLabel        = "os"
	OSVolume       = "os"
	EtcVolume      = "etc"
	EtcPath        = "/etc"
	SnapshotLabel  = "snapshots"
	SnapshotVolume = "snapshots"
)

func NewGroup(group *types.DiskGroup, dest string) (*Group, error) {
	if group.Stage != types.Stage1 {
		return nil, errors.Errorf("unsupported stage %s", group.Stage)
	}
	for _, s := range group.Subvolumes {
		if s.Path == EtcPath {
			return nil, errors.Errorf("user managed etc subvol not allowed %s", s.Name)
		}
	}
	return &Group{
		group: group,
		dest:  dest,
	}, nil
}

type Group struct {
	group  *types.DiskGroup
	mounts []string
	dest   string
}

type Mount struct {
	Source      string
	Destination string
}

func (d *Group) String() string {
	return d.group.Label
}

type InitConfig struct {
	AdditionalMounts []Mount
	EtcSubvolume     bool
}

// Init the entire group returning the path to access the group
func (d *Group) Init(diskMount string, config *InitConfig) error {
	d.mounts = append(d.mounts, diskMount)

	// create subvolumes
	subvolumes := d.group.Subvolumes
	if d.group.Label == OSLabel {
		osVolumes := []*types.Subvolume{
			{
				Name: OSVolume,
				Path: "/",
			},
		}
		if config != nil && config.EtcSubvolume {
			osVolumes = append(osVolumes, &types.Subvolume{
				Name: EtcVolume,
				Path: EtcPath,
			})
		}
		subvolumes = append(osVolumes, subvolumes...)
	}
	if err := btrfs.CreateSubvolumes(diskMount, subvolumes); err != nil {
		return errors.Wrap(err, "create subvolumes")
	}
	for _, s := range subvolumes {
		dest := filepath.Join(d.dest, s.Path)
		if err := os.MkdirAll(dest, 0711); err != nil {
			return errors.Wrapf(err, "mkdir subvolumes %s", dest)
		}
		if err := unix.Mount(filepath.Join(diskMount, s.Name), dest, "none", unix.MS_BIND, ""); err != nil {
			return errors.Wrapf(err, "mount subvolume %s", s.Name)
		}
		d.mounts = append(d.mounts, dest)
	}
	if config != nil {
		for _, m := range config.AdditionalMounts {
			dest := filepath.Join(d.dest, m.Destination)
			if err := os.MkdirAll(dest, 0711); err != nil {
				return errors.Wrapf(err, "mkdir additional mount %s", dest)
			}
			if err := unix.Mount(m.Source, dest, "none", unix.MS_BIND, ""); err != nil {
				return errors.Wrapf(err, "mount additional mount %s", m.Source)
			}
			d.mounts = append(d.mounts, dest)
		}
	}
	return nil
}

// Close the group
func (d *Group) Close() error {
	var last string
	for _, path := range reverse(d.mounts) {
		if err := unix.Unmount(path, 0); err != nil {
			return errors.Wrapf(err, "unmount %s", path)
		}
		last = path
	}
	return os.RemoveAll(last)
}

// Entries returns the fstab entries for the group
func (d *Group) Entries() []*fstab.Entry {
	var entries []*fstab.Entry
	for _, s := range d.group.Subvolumes {
		options := []string{
			fmt.Sprintf("subvol=/%s", s.Name),
		}
		if !s.Cow {
			options = append(options, "nodatacow")
		}
		entries = append(entries, &fstab.Entry{
			Type:    mkfs.Btrfs,
			Device:  fmt.Sprintf("LABEL=%s", d.group.Label),
			Path:    s.Path,
			Pass:    2,
			Options: options,
		})
	}
	return entries
}

func reverse(mounts []string) []string {
	for i, j := 0, len(mounts)-1; i < j; i, j = i+1, j-1 {
		mounts[i], mounts[j] = mounts[j], mounts[i]
	}
	return mounts
}
