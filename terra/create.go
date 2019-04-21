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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const serverTemplate = `# syntax=docker/dockerfile:experimental

{{range $v := .Imports -}}
FROM {{imageName $v}} as {{cname $v}}
{{end}}

FROM {{.Base}}

{{.Userland}}

{{range $v := .Imports -}}
{{if ne $v.Name "kernel"}}
COPY --from={{cname $v}} / /
{{if $v.Systemd}}RUN systemctl enable {{cname $v}}{{end}}{{end}}
{{end}}

ADD hostname /etc/hostname
ADD hosts /etc/hosts

RUN mkdir -p /home/terra/.ssh
ADD keys /home/terra/.ssh/authorized_keys
RUN chown -R terra:terra /home/terra

{{if .Init}}CMD ["{{.Init}}"]{{end}}
`

type ServerConfig struct {
	Version    string       `toml:"version"`
	OS         string       `toml:"os"`
	Hostname   string       `toml:"hostname"`
	Components []*Component `toml:"components"`
	Userland   string       `toml:"userland"`
	Init       string       `toml:"init"`
	SSH        SSH          `toml:"ssh"`
}

type SSH struct {
	Github string `toml:"github"`
}

func loadServerConfig(path string) (*ServerConfig, error) {
	var c ServerConfig
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	if c.OS == "" {
		return nil, errors.New("no os defined")
	}
	for _, i := range c.Components {
		if i.Version == "" {
			i.Version = defaultVersion
		}
	}
	return &c, nil
}

const hostsTemplate = `127.0.0.1       localhost %s
::1             localhost ip6-localhost ip6-loopback
ff02::1         ip6-allnodes
ff02::2         ip6-allrouters`

func setupHostname(path, hostname string) error {
	if hostname == "" {
		hostname = "terra"
	}
	f, err := os.Create(filepath.Join(path, "hostname"))
	if err != nil {
		return err
	}
	_, err = f.WriteString(hostname)
	f.Close()
	if err != nil {
		return err
	}
	if f, err = os.Create(filepath.Join(path, "hosts")); err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, hostsTemplate, hostname)
	f.Close()
	return err
}

func setupSSH(path string, ssh SSH) error {
	if ssh.Github != "" {
		r, err := http.Get(fmt.Sprintf("https://github.com/%s.keys", ssh.Github))
		if err != nil {
			return err
		}
		defer r.Body.Close()
		f, err := os.Create(filepath.Join(path, "keys"))
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(f, r.Body); err != nil {
			return err
		}
	}
	return nil
}

var createCommand = cli.Command{
	Name:  "create",
	Usage: "create new server image",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "context,c",
			Usage: "specify the context path",
			Value: ".",
		},
		cli.StringFlag{
			Name:  "repo",
			Usage: "set the image repository",
		},
	},
	Action: func(clix *cli.Context) error {
		config, err := loadServerConfig(clix.Args().First())
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(clix.String("context"))
		if err != nil {
			return err
		}
		ctx := cancelContext()
		osCtx := &OSContext{
			Base:     config.OS,
			Userland: config.Userland,
			Init:     config.Init,
			Hostname: config.Hostname,
		}
		for _, c := range config.Components {
			osCtx.Imports = append(osCtx.Imports, c)
		}
		path, err := writeDockerfile(osCtx, serverTemplate)
		if err != nil {
			return err
		}
		defer os.RemoveAll(path)

		if err := setupHostname(abs, config.Hostname); err != nil {
			return err
		}
		if err := setupSSH(abs, config.SSH); err != nil {
			return err
		}

		cmd := exec.CommandContext(ctx, "vab", "build",
			"-c", abs,
			"--ref", fmt.Sprintf("%s:%s", clix.String("repo"), config.Version),
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
