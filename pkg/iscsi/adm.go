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
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func Adm(ctx context.Context, args ...string) error {
	out, err := exec.CommandContext(ctx, "iscsiadm", args...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func Discover(ctx context.Context, portal string) error {
	if err := Adm(ctx,
		"--mode", "discovery",
		"-t", "sendtargets",
		"--portal", portal); err != nil {
		return errors.Wrap(err, "discover targets")
	}
	return nil
}

type Target struct {
	portal string
	iqn    string

	path string
}

func LoadTarget(portal, iqn string) *Target {
	return &Target{
		portal: portal,
		iqn:    iqn,
		path:   path(portal, iqn),
	}
}

func (t *Target) Path() string {
	return t.path
}

func (t *Target) Lun(i int) string {
	return fmt.Sprintf("%s-lun-%d", t.path, i)
}

func (t *Target) Partition(lun, part int) string {
	return fmt.Sprintf("%s-part%d", t.Lun(lun), part)
}

func (t *Target) Logout(ctx context.Context) error {
	return Logout(ctx, t.portal, t.iqn)
}

func (t *Target) Ready(d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	path := t.Lun(0)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if _, err := os.Lstat(path); err == nil {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func Login(ctx context.Context, portal, iqn string) (*Target, error) {
	if err := Adm(ctx,
		"--mode", "node",
		"--targetname", iqn,
		"--portal", portal,
		"--login"); err != nil && !isPresentErr(err) {
		return nil, errors.Wrap(err, "login")
	}
	return &Target{
		portal: portal,
		iqn:    iqn,
		path:   path(portal, iqn),
	}, nil
}

func Logout(ctx context.Context, portal, iqn string) error {
	if err := Adm(ctx,
		"--mode", "node",
		"--targetname", iqn,
		"--portal", portal,
		"--logout"); err != nil && !isNotSessionErr(err) {
		return errors.Wrap(err, "logout")
	}
	return nil
}

func path(portal, iqn string) string {
	return filepath.Join("/dev/disk/by-path", fmt.Sprintf("ip-%s:3260-iscsi-%s", portal, iqn))
}

func isPresentErr(err error) bool {
	return strings.Contains(err.Error(), "already present")
}

func isNotSessionErr(err error) bool {
	return strings.Contains(err.Error(), "No matching sessions found")
}
