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

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const devicePath = "/sd"

func disk(args ...string) string {
	return filepath.Join(append([]string{devicePath}, args...)...)
}

// before mounts the device before doing operations
func before(clix *cli.Context) error {
	if err := os.MkdirAll(devicePath, 0755); err != nil {
		return err
	}
	return syscall.Mount(clix.String("device"), devicePath, clix.String("fs-type"), 0, "")
}

// after unmounts the device
func after(clix *cli.Context) error {
	for i := 0; i < 30; i++ {
		if err := syscall.Unmount(devicePath, 0); err == nil {
			return err
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("unable to remove mount")
}

func writeDockerfile(ctx *OSContext, tmpl string) (string, error) {
	tmp, err := ioutil.TempDir("", "osb-")
	if err != nil {
		return "", err
	}
	f, err := os.Create(filepath.Join(tmp, "Dockerfile"))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := render(f, tmpl, ctx); err != nil {
		return "", err
	}
	return tmp, nil
}
