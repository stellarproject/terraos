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
	"path/filepath"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

const (
	defaultRepo = "stellarproject"
)

const serverTemplate = `# syntax=docker/dockerfile:experimental

{{range $v := .Imports -}}
FROM {{imageName $v}} as {{cname $v}}
{{end}}

FROM {{.Base}}

{{range $v := .Imports -}}
{{if ne $v.Name "kernel"}}
COPY --from={{cname $v}} / /
{{range $s := $v.Systemd}}
RUN systemctl enable {{$s}}
{{end}}
{{end}}
{{end}}

ADD hostname /etc/hostname
ADD hosts /etc/hosts
ADD 01-netcfg.yaml /etc/netplan/

RUN mkdir -p /home/terra/.ssh
ADD keys /home/terra/.ssh/authorized_keys
RUN chown -R terra:terra /home/terra
RUN dbus-uuidgen --ensure=/etc/machine-id && dbus-uuidgen --ensure

{{.Userland}}

{{if .Init}}CMD ["{{.Init}}"]{{end}}
`
const hostsTemplate = `127.0.0.1       localhost %s
::1             localhost ip6-localhost ip6-loopback
ff02::1         ip6-allnodes
ff02::2         ip6-allrouters`

type Component struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Systemd []string `toml:"systemd"`
}

type OSContext struct {
	Base     string
	Userland string
	Imports  []*Component
	Kernel   string
	Init     string
	Hostname string
}

type ServerConfig struct {
	ID         string       `toml:"id"`
	Version    string       `toml:"version"`
	Repo       string       `toml:"repo"`
	OS         string       `toml:"os"`
	Components []*Component `toml:"components"`
	Userland   string       `toml:"userland"`
	Init       string       `toml:"init"`
	SSH        SSH          `toml:"ssh"`
	PXE        *PXE         `toml:"pxe"`
	Netplan    Netplan      `toml:"netplan"`
	FS         FS           `toml:"fs"`
}

type SSH struct {
	Github string `toml:"github"`
}

type FS struct {
	Type   string   `toml:"type"`
	RWDirs []string `toml:"rw_dirs"`
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

func loadServerConfig(path string) (*ServerConfig, error) {
	var c ServerConfig
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	if c.Version == "" {
		return nil, errors.New("no version specified")
	}
	if c.OS == "" {
		return nil, errors.New("no os defined")
	}
	for _, i := range c.Components {
		if i.Version == "" {
			i.Version = c.Version
		}
	}
	return &c, nil
}

func setupHostname(path, hostname string, paths *[]string) error {
	if hostname == "" {
		hostname = "terra"
	}
	f, err := os.Create(filepath.Join(path, "hostname"))
	if err != nil {
		return err
	}
	*paths = append(*paths, f.Name())
	_, err = f.WriteString(hostname)
	f.Close()
	if err != nil {
		return err
	}
	if f, err = os.Create(filepath.Join(path, "hosts")); err != nil {
		return err
	}
	*paths = append(*paths, f.Name())
	_, err = fmt.Fprintf(f, hostsTemplate, hostname)
	f.Close()
	return err
}

func setupSSH(path string, ssh SSH, paths *[]string) error {
	p := filepath.Join(path, "keys")
	*paths = append(*paths, p)
	if ssh.Github != "" {
		r, err := http.Get(fmt.Sprintf("https://github.com/%s.keys", ssh.Github))
		if err != nil {
			return err
		}
		defer r.Body.Close()
		f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(f, r.Body); err != nil {
			return err
		}
	} else {
		f, err := os.OpenFile(filepath.Join(path, "keys"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}
