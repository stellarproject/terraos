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
	"text/template"

	"github.com/BurntSushi/toml"
)

const (
	defaultRepo      = "docker.io/stellarproject"
	defaultBaseImage = "docker.io/stellarproject/ubuntu:18.10"
	defaultVersion   = "latest"
)

type Component struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Systemd bool   `toml:"systemd"`
}

type Config struct {
	Version    string       `toml:"version"`
	Base       string       `toml:"base"`
	Kernel     string       `toml:"kernel"`
	Components []*Component `toml:"components"`
	Userland   string       `toml:"userland"`
	Init       string       `toml:"init"`
}

func loadConfig(path string) (*Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	if c.Base == "" {
		c.Base = defaultBaseImage
	}
	for _, i := range c.Components {
		if i.Version == "" {
			i.Version = defaultVersion
		}
	}
	return &c, nil
}

func joinImage(i, name, version string) string {
	return fmt.Sprintf("%s/%s:%s", i, name, version)
}

type OSContext struct {
	Base     string
	Userland string
	Imports  []*Component
	Kernel   string
	Init     string
}

func (o *OSContext) GetBase() string {
	if o.Base == "" {
		o.Base = defaultBaseImage
	}
	return o.Base
}

func cname(c Component) string {
	return c.Name
}

func imageName(c Component) string {
	return joinImage(defaultRepo, c.Name, c.Version)
}

func render(w io.Writer, ctx *OSContext) error {
	t, err := template.New("dockerfile").Funcs(template.FuncMap{
		"cname":     cname,
		"imageName": imageName,
	}).Parse(osTemplate)
	if err != nil {
		return err
	}
	return t.Execute(w, ctx)
}

const osTemplate = `# syntax=docker/dockerfile:experimental

{{range $v := .Imports -}}
FROM {{imageName $v}} as {{cname $v}}
{{end}}

FROM {{.Base}}

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-{{.Kernel}}-terra_{{.Kernel}}-terra-1_amd64.deb \
	/tmp/linux-libc-dev_{{.Kernel}}-terra-1_amd64.deb

{{.Userland}}

{{range $v := .Imports -}}
{{if ne $v.Name "kernel"}}
COPY --from={{cname $v}} / /
{{if $v.Systemd}}RUN systemctl enable {{cname $v}}{{end}}{{end}}
{{end}}

{{if .Init}}CMD ["{{.Init}}"]{{end}}
`
