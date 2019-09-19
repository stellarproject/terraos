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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func git(ctx context.Context, args ...string) error {
	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func commit(ctx context.Context, message string) error {
	return git(ctx, "commit", "-m", fmt.Sprintf("terra: %s", message))
}

func createTerraClusterFile() error {
	f, err := os.Create(clusterFilename)
	if err != nil {
		return err
	}
	return f.Close()
}

func writeFile(name, contents string) error {
	return ioutil.WriteFile(name, []byte(contents), 0644)
}

func createDirs(dirs []string, gitkeep bool) error {
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return errors.Wrapf(err, "create dir %s", d)
		}
		if gitkeep {
			f, err := os.Create(filepath.Join(d, ".gitkeep"))
			if err != nil {
				return errors.Wrap(err, "unable to create .gitkeep")
			}
			f.Close()
		}
	}
	return nil
}

func allocateIP(path, address string) (int, string) {
	parts := strings.Split(address, ".")
	for i := 2; i < 254; i++ {
		parts[3] = strconv.Itoa(i)
		ip := strings.Join(parts, ".")
		if err := os.Mkdir(filepath.Join(path, ip), 0755); err != nil {
			continue
		}
		return i, ip
	}
	return -1, ""
}
