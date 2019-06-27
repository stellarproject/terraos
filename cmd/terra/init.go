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
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/urfave/cli"
)

const service = `[Unit]
Description=terra controller
After=network.target

[Service]
ExecStart=/usr/local/bin/terra-controller
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target`

var initCommand = cli.Command{
	Name:        "init",
	Description: "init a node to run the controller",
	Action: func(clix *cli.Context) error {
		if err := os.MkdirAll("/cluster", 0755); err != nil {
			return err
		}
		if err := writeConfig(); err != nil {
			return err
		}
		if err := writeService(); err != nil {
			return err
		}
		return enableService()
	},
}

func enableService() error {
	if err := systemd("enable", "terra-controller"); err != nil {
		return err
	}
	return systemd("start", "terra-controller")
}

func systemd(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func writeService() error {
	f, err := os.Create(filepath.Join("/etc/systemd/system/terra-controller.service"))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(service)
	return err
}

func writeConfig() error {
	config := cmd.Default()
	f, err := os.Create(cmd.ConfigPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return config.Write(f)
}
