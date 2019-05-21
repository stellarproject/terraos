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

package btrfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

type Subvolume struct {
	Name string
	Path string
}

func CreateSubvolumes(subvolumes []Subvolume, path string) error {
	for _, s := range subvolumes {
		sv := filepath.Join(path, s.Name)
		if _, err := os.Stat(sv); err == nil {
			continue
		}
		if err := btrfs("subvolume", "create", sv); err != nil {
			return errors.Wrap(err, "create sub volume")
		}
	}
	return nil
}

func OverlaySubvolumes(root string, subvolumes []Subvolume, deviceMount string) (paths []string, err error) {
	defer func() {
		if err != nil {
			for _, p := range paths {
				syscall.Unmount(p, 0)
			}
		}
	}()
	for _, s := range subvolumes {
		subPath := filepath.Join(root, s.Path)
		if err := os.MkdirAll(subPath, 0755); err != nil {
			return nil, errors.Wrapf(err, "mkdir subvolume %s", subPath)
		}
		if err := syscall.Mount(filepath.Join(deviceMount, s.Name), subPath, "none", syscall.MS_BIND, ""); err != nil {
			return nil, errors.Wrapf(err, "mount %s:%s", s.Name, subPath)
		}
		paths = append(paths, subPath)
	}
	// add the root path
	paths = append(paths, root)
	return paths, nil
}

func MountSubvolumes(subvolumes map[string]string, device, path string) (paths []string, err error) {
	defer func() {
		if err != nil {
			for _, p := range paths {
				syscall.Unmount(p, 0)
			}
		}
	}()
	for k, v := range subvolumes {
		subPath := filepath.Join(path, v)
		if err := os.MkdirAll(subPath, 0755); err != nil {
			return nil, errors.Wrapf(err, "mkdir subvolume %s", subPath)
		}
		if err := syscall.Mount(device, subPath, "btrfs", 0, fmt.Sprintf("subvol=%s", k)); err != nil {
			return nil, errors.Wrapf(err, "mount %s:%s", k, subPath)
		}
		paths = append(paths, subPath)
	}
	paths = append(paths, path)
	return paths, nil
}

func btrfs(args ...string) error {
	out, err := exec.Command("btrfs", args...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
