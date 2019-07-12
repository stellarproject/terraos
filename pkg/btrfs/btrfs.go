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
	"os/exec"

	"github.com/pkg/errors"
)

func Check() error {
	if _, err := exec.LookPath("btrfs"); err != nil {
		return errors.Wrap(err, "btrfs command cannot be found")
	}
	return nil
}

func CreateSubvolume(path string) error {
	if err := Btrfs("subvolume", "create", path); err != nil {
		return errors.Wrapf(err, "create subvolume %s", path)
	}
	return nil
}

func Snapshot(source, dest string) error {
	if err := Btrfs("subvolume", "snapshot", source, dest); err != nil {
		return errors.Wrapf(err, "create snapshot of %s to %s", source, dest)
	}
	return nil
}

func Btrfs(args ...string) error {
	out, err := exec.Command("btrfs", args...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
