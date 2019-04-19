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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/urfave/cli"
)

var osCommand = cli.Command{
	Name:  "os",
	Usage: "build a os new release",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "context,c",
			Usage: "specify the context path",
			Value: ".",
		},
	},
	Action: func(clix *cli.Context) error {
		config, err := loadConfig(clix.Args().First())
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(clix.String("context"))
		if err != nil {
			return err
		}
		ctx := cancelContext()
		osCtx := &OSContext{
			Kernel:   config.Kernel,
			Base:     config.Base,
			Userland: config.Userland,
			Init:     config.Init,
			Imports: []*Component{
				{
					Name:    "kernel",
					Version: config.Kernel,
				},
			},
		}
		for _, c := range config.Components {
			osCtx.Imports = append(osCtx.Imports, c)
		}
		path, err := writeDir(osCtx)
		if err != nil {
			return err
		}
		defer os.RemoveAll(path)

		cmd := exec.CommandContext(ctx, "vab", "build",
			"-c", abs,
			"--ref", fmt.Sprintf("%s/terraos:%s", defaultRepo, config.Version),
			"--push",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = path

		if err := cmd.Run(); err != nil {
			f, ferr := os.Open(filepath.Join(path, "Dockerfile"))
			if ferr != nil {
				return ferr
			}
			defer f.Close()
			io.Copy(os.Stdout, f)
			return err
		}
		return nil // return createISO(ctx, config)
	},
}

func createISO(ctx context.Context, config *Config) error {
	cmd := exec.CommandContext(ctx, "vab", "build",
		"--arg", fmt.Sprintf("KERNEL_VERSION=%s", config.Kernel),
		"-c", "iso",
		"-d", "iso",
		"--local",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeDir(ctx *OSContext) (string, error) {
	tmp, err := ioutil.TempDir("", "osb-")
	if err != nil {
		return "", err
	}
	f, err := os.Create(filepath.Join(tmp, "Dockerfile"))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := render(f, ctx); err != nil {
		return "", err
	}
	return tmp, nil
}
