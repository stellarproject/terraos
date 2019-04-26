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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
)

const (
	defaultRepo      = "docker.io/stellarproject"
	defaultBaseImage = "docker.io/stellarproject/ubuntu:18.10"
)

type OSContext struct {
	Base     string
	Userland string
	Imports  []*Component
	Kernel   string
	Init     string
	Hostname string
}

const osTemplate = `# syntax=docker/dockerfile:experimental

{{range $v := .Imports -}}
FROM {{imageName $v}} as {{cname $v}}
{{end}}

FROM {{.Base}}

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-{{.Kernel}}-terra_{{.Kernel}}-terra-1_amd64.deb \
	/tmp/linux-image-{{.Kernel}}-terra_{{.Kernel}}-terra-1_amd64.deb \
	/tmp/linux-libc-dev_{{.Kernel}}-terra-1_amd64.deb && \
	cp /tmp/wg /usr/local/bin/

{{.Userland}}

{{range $v := .Imports -}}
{{if ne $v.Name "kernel"}}
COPY --from={{cname $v}} / /
{{if $v.Systemd}}RUN systemctl enable {{cname $v}}{{end}}{{end}}
{{end}}

{{if .Init}}CMD ["{{.Init}}"]{{end}}
`

type Component struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Systemd bool   `toml:"systemd"`
}

type OSConfig struct {
	Version    string       `toml:"version"`
	Base       string       `toml:"base"`
	Kernel     string       `toml:"kernel"`
	Components []*Component `toml:"components"`
	Userland   string       `toml:"userland"`
	Init       string       `toml:"init"`
}

func loadOSConfig(path string) (*OSConfig, error) {
	var c OSConfig
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	if c.Version == "" {
		return nil, errors.New("no version specified")
	}
	if c.Base == "" {
		c.Base = defaultBaseImage
	}
	if c.Init == "" {
		c.Init = "/sbin/init"
	}
	for _, i := range c.Components {
		if i.Version == "" {
			i.Version = c.Version
		}
	}
	return &c, nil
}

func joinImage(i, name, version string) string {
	return fmt.Sprintf("%s/%s:%s", i, name, version)
}

func cname(c Component) string {
	return c.Name
}

func cmdargs(args []string) string {
	return strings.Join(args, " ")
}

func imageName(c Component) string {
	return joinImage(defaultRepo, c.Name, c.Version)
}

func render(w io.Writer, tmp string, ctx *OSContext) error {
	t, err := template.New("dockerfile").Funcs(template.FuncMap{
		"cname":     cname,
		"imageName": imageName,
		"cmdargs":   cmdargs,
	}).Parse(tmp)
	if err != nil {
		return err
	}
	return t.Execute(w, ctx)
}

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
		config, err := loadOSConfig(clix.Args().First())
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
		path, err := writeDockerfile(osCtx, osTemplate)
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
		return nil
	},
}
