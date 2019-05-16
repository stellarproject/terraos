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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "terra-create"
	app.Version = version.Version
	app.Usage = "[file.toml]"
	app.Description = `
                                                     ___
                                                  ,o88888
                                               ,o8888888'
                         ,:o:o:oooo.        ,8O88Pd8888"
                     ,.::.::o:ooooOoOoO. ,oO8O8Pd888'"
                   ,.:.::o:ooOoOoOO8O8OOo.8OOPd8O8O"
                  , ..:.::o:ooOoOOOO8OOOOo.FdO8O8"
                 , ..:.::o:ooOoOO8O888O8O,COCOO"
                , . ..:.::o:ooOoOOOO8OOOOCOCO"
                 . ..:.::o:ooOoOoOO8O8OCCCC"o
                    . ..:.::o:ooooOoCoCCC"o:o
                    . ..:.::o:o:,cooooCo"oo:o:
                 ` + "`" + `   . . ..:.:cocoooo"'o:o:::'
                 .` + "`" + `   . ..::ccccoc"'o:o:o:::'
                :.:.    ,c:cccc"':.:.:.:.:.'
              ..:.:"'` + "`" + `::::c:"'..:.:.:.:.:.'
            ...:.'.:.::::"'    . . . . .'
           .. . ....:."' ` + "`" + `   .  . . ''
         . . . ...."'
         .. . ."'
        .
Terra OS management`
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.StringFlag{
			Name:  "context,c",
			Usage: "specify the context path",
			Value: ".",
		},
		cli.BoolFlag{
			Name:  "push",
			Usage: "push the resulting image",
		},
		cli.StringFlag{
			Name:  "userland,u",
			Usage: "userland file path",
		},
	}
	app.Before = func(clix *cli.Context) error {
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Action = func(clix *cli.Context) error {
		config, err := loadServerConfig(clix.Args().First())
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(clix.String("context"))
		if err != nil {
			return err
		}
		var (
			paths []string
			ctx   = cmd.CancelContext()
		)
		osCtx := &OSContext{
			Base:       config.OS,
			Userland:   config.Userland,
			Init:       config.Init,
			Hostname:   config.ID,
			ResolvConf: len(config.Nameservers) > 0,
		}
		if userland := clix.GlobalString("userland"); userland != "" {
			data, err := ioutil.ReadFile(userland)
			if err != nil {
				return err
			}
			osCtx.Userland = string(data)
		}
		defer func() {
			for _, p := range paths {
				os.Remove(p)
			}
		}()
		for _, c := range config.Components {
			osCtx.Imports = append(osCtx.Imports, c)
		}
		path, err := writeDockerfile(osCtx, serverTemplate)
		if err != nil {
			return err
		}
		defer os.RemoveAll(path)

		if err := setupHostname(abs, config.ID, &paths); err != nil {
			return err
		}
		if err := setupSSH(abs, config.SSH, &paths); err != nil {
			return err
		}
		if err := setupNetplan(abs, config.Netplan, &paths); err != nil {
			return err
		}
		if err := setupFstab(abs, config.FS, &paths); err != nil {
			return err
		}
		if err := setupResolvConf(abs, config.Nameservers, &paths); err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "vab", "build",
			"-c", abs,
			"--ref", fmt.Sprintf("%s/%s:%s", config.Repo, config.ID, config.Version),
			"--push="+strconv.FormatBool(clix.Bool("push")),
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
		if config.PXE != nil {
			return setupPXE(config.ID, config.Version, config.FS, config.PXE)
		}
		return nil

	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupFstab(path string, fs FS, paths *[]string) error {
	f, err := os.Create(filepath.Join(path, "fstab"))
	if err != nil {
		return err
	}
	*paths = append(*paths, f.Name())
	defer f.Close()
	if fs.Type != "btrfs" {
		return nil
	}
	entries := []*fstab.Entry{
		{
			Device: "LABEL=terra",
			Path:   "/home",
			Type:   "btrfs",
			Options: []string{
				"subvol=/home",
			},
			Pass: 2,
		},
		{
			Device: "LABEL=terra",
			Path:   "/var/log",
			Type:   "btrfs",
			Options: []string{
				"subvol=/log",
			},
			Pass: 2,
		},
		{
			Device: "LABEL=terra",
			Path:   "/var/lib/containerd",
			Type:   "btrfs",
			Options: []string{
				"subvol=/containerd",
			},
			Pass: 2,
		},
	}
	return fstab.Write(f, entries)
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
