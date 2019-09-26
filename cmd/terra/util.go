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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/containerd/containerd/content"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/cluster/v1"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/urfave/cli"
)

func removePartition(device string) string {
	partition := string(device[len(device)-1])
	if _, err := strconv.Atoi(partition); err != nil {
		return device
	}
	if strings.Contains(device, "nvme") {
		partition = "p" + partition
	}
	return strings.TrimSuffix(device, partition)
}

func getDevices(clix *cli.Context) (map[string]string, error) {
	var (
		out = make(map[string]string)
	)
	for _, d := range clix.StringSlice("device") {
		parts := strings.Split(d, ":")
		if len(parts) != 2 {
			return nil, errors.Errorf("device %s not valid format", d)
		}
		out[parts[0]] = parts[1]
	}
	return out, nil
}

func checkYes(msg string, scanner *bufio.Scanner) error {
	fmt.Fprint(os.Stderr, msg)
	fmt.Fprintln(os.Stderr, " [y/n]")
	if !scanner.Scan() {
		return errors.New("no input")
	}
	if strings.ToLower(scanner.Text()) == "y" {
		return nil
	}
	return errors.New("user aborted")
}

func getStore() (content.Store, error) {
	return image.NewContentStore(contentStorePath)
}

func getCluster(clix *cli.Context) *v1.Store {
	return v1.New(clix.GlobalString("cluster"), "")
}

func tmpContentStore() (content.Store, func() error, error) {
	dir, err := ioutil.TempDir("/tmp", "content-")
	if err != nil {
		return nil, nil, err
	}
	s, err := image.NewContentStore(dir)
	if err != nil {
		os.RemoveAll(dir)
		return nil, nil, err
	}
	return s, func() error {
		return os.RemoveAll(dir)
	}, nil
}
