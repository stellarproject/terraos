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
	"os/exec"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/urfave/cli"
)

var iscsiCommand = cli.Command{
	Name:        "iscsi",
	Description: "manage iscsi targets",
	Subcommands: []cli.Command{
		iscsiLoginCommand,
		iscsiLogoutCommand,
	},
}

var iscsiLoginCommand = cli.Command{
	Name:        "login",
	Description: "login a node's iscsi target to the local system",
	ArgsUsage:   "[node.toml]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "iscsi-target,target",
			Usage:  "iscsi target IP",
			EnvVar: "ISCSI_TARGET",
		},
	},
	Action: func(clix *cli.Context) error {
		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		ctx := cmd.CancelContext()
		if err := iscsiadm(ctx,
			"--mode", "discovery",
			"-t", "sendtargets",
			"--portal", clix.String("target")); err != nil {
			return errors.Wrap(err, "discover targets")
		}
		for _, v := range node.Volumes {
			if v.IsISCSI() {
				if err := iscsiadm(ctx,
					"--mode", "node",
					"--targetname", v.TargetIqn,
					"--portal", clix.String("target"),
					"--login"); err != nil {
					return errors.Wrap(err, "login")
				}
				return nil
			}
		}
		return nil
	},
}

var iscsiLogoutCommand = cli.Command{
	Name:        "logout",
	Description: "logout a node's iscsi target to the local system",
	ArgsUsage:   "[node.toml]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "iscsi-target,target",
			Usage:  "iscsi target IP",
			EnvVar: "ISCSI_TARGET",
		},
	},
	Action: func(clix *cli.Context) error {
		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		ctx := cmd.CancelContext()
		for _, v := range node.Volumes {
			if v.IsISCSI() {
				if err := iscsiadm(ctx,
					"--mode", "node",
					"--targetname", v.TargetIqn,
					"--portal", clix.String("target"),
					"--logout"); err != nil {
					return errors.Wrap(err, "logout")
				}
				return nil
			}
		}
		return nil
	},
}

func iscsiadm(ctx context.Context, args ...string) error {
	out, err := exec.CommandContext(ctx, "iscsiadm", args...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
