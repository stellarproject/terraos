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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	v1 "github.com/stellarproject/terraos/config/v1"
	"github.com/stellarproject/terraos/pkg/netplan"
	"github.com/stellarproject/terraos/pkg/resolvconf"
	"github.com/urfave/cli"
)

var createCommand = cli.Command{
	Name:  "create",
	Usage: "create a new machine image",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "context,c",
			Usage: "specify the context path",
			Value: ".",
		},
		cli.BoolFlag{
			Name:  "push",
			Usage: "push the resulting image",
		},
		cli.BoolFlag{
			Name:  "http",
			Usage: "push the image over http",
		},
		cli.StringFlag{
			Name:  "userland,u",
			Usage: "userland file path",
		},
		cli.BoolFlag{
			Name:  "dump",
			Usage: "dump and example config",
		},
		cli.BoolFlag{
			Name:  "dry,n",
			Usage: "dry run without building",
		},
		cli.BoolFlag{
			Name:  "no-cache",
			Usage: "build with no cache",
		},
		cli.StringFlag{
			Name:  "vhost",
			Usage: "file to output vhost config",
		},
	},
	Action: func(clix *cli.Context) error {
		if clix.Bool("dump") {
			return dumpConfig()
		}
		config, err := loadServerConfig(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load server toml")
		}
		abs, err := filepath.Abs(clix.String("context"))
		if err != nil {
			return errors.Wrap(err, "context absolute path")
		}
		var (
			paths []string
			ctx   = cmd.CancelContext()
		)
		imageContext := &ImageContext{
			Base:     config.OS,
			Userland: config.Userland,
			Init:     config.Init,
			Hostname: config.Hostname,
		}
		if userland := clix.String("userland"); userland != "" {
			data, err := ioutil.ReadFile(userland)
			if err != nil {
				return errors.Wrapf(err, "read userland file %s", userland)
			}
			imageContext.Userland = string(data)
		}
		if !clix.Bool("dry") {
			defer func() {
				for _, p := range paths {
					os.Remove(p)
				}
			}()
		}
		for _, c := range config.Components {
			imageContext.Imports = append(imageContext.Imports, c)
		}
		path, err := writeDockerfile(imageContext, serverTemplate)
		if err != nil {
			return errors.Wrap(err, "write dockerfile")
		}

		if !clix.Bool("dry") {
			defer os.RemoveAll(path)
		}
		if clix.Bool("dry") {
			return nil
		}
		ref := fmt.Sprintf("%s/%s:%s", config.Repo, config.Hostname, config.Version)
		cmd := exec.CommandContext(ctx, "vab", "build",
			"-c", abs,
			"--ref", ref,
			"--push="+strconv.FormatBool(clix.Bool("push")),
			"--no-cache="+strconv.FormatBool(clix.Bool("no-cache")),
			"--http="+strconv.FormatBool(clix.Bool("http")),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = path

		if err := cmd.Run(); err != nil {
			f, ferr := os.Open(filepath.Join(path, "Dockerfile"))
			if ferr != nil {
				return errors.Wrap(ferr, "open Dockerfile")
			}
			defer f.Close()
			io.Copy(os.Stdout, f)
			return errors.Wrap(err, "execute build")
		}
		if vhost := clix.String("vhost"); vhost != "" {
			c := &v1.Container{
				ID:         fmt.Sprintf("%s-vhost", config.Hostname),
				Image:      ref,
				Privileged: true,
				MaskedPaths: []string{
					"/etc/netplan",
				},
				Networks: []*v1.Network{
					{
						Type: "macvlan",
						Name: "vhost0",
						IPAM: v1.IPAM{
							Type: "dhcp",
						},
					},
				},
				Resources: &v1.Resources{
					CPU:    1.0,
					Memory: 128,
				},
			}

			f, err := os.Create(vhost)
			if err != nil {
				return errors.Wrapf(err, "create vhost file %s", vhost)
			}
			defer f.Close()

			if err := toml.NewEncoder(f).Encode(c); err != nil {
				return errors.Wrap(err, "write vhost config")
			}
		}
		return nil
	},
}

func writeDockerfile(ctx *ImageContext, tmpl string) (string, error) {
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

const serverTemplate = `# syntax=docker/dockerfile:experimental

{{range $v := .Imports -}}
FROM {{imageName $v}} as {{cname $v}}
{{end}}

FROM {{.Base}}

{{range $v := .Imports -}}
COPY --from={{cname $v}} / /
{{range $s := $v.Systemd}}
RUN systemctl enable {{$s}}
{{end}}
{{end}}

RUN dbus-uuidgen --ensure=/etc/machine-id && dbus-uuidgen --ensure

{{.Userland}}

{{if .Init}}CMD ["{{.Init}}"]{{end}}
`

func cname(c Component) string {
	h := md5.New()
	h.Write([]byte(c.Image))
	return "I" + hex.EncodeToString(h.Sum(nil))
}

func cmdargs(args []string) string {
	return strings.Join(args, " ")
}

func imageName(c Component) string {
	return c.Image
}

func render(w io.Writer, tmp string, ctx *ImageContext) error {
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
	return &c, nil
}

type Component struct {
	Image   string   `toml:"image"`
	Systemd []string `toml:"systemd"`
}

type ImageContext struct {
	Base       string
	Userland   string
	Imports    []*Component
	Kernel     string
	Init       string
	Hostname   string
	ResolvConf bool
}

type ServerConfig struct {
	Hostname   string           `toml:"hostname"`
	Version    string           `toml:"version"`
	Repo       string           `toml:"repo"`
	OS         string           `toml:"os"`
	Components []*Component     `toml:"components"`
	Userland   string           `toml:"userland"`
	Init       string           `toml:"init"`
	SSH        SSH              `toml:"ssh"`
	Netplan    *netplan.Netplan `toml:"netplan"`
	ResolvConf *resolvconf.Conf `toml:"resolvconf"`
}

type SSH struct {
	Github string `toml:"github"`
}

func dumpConfig() error {
	c := &ServerConfig{
		Hostname: "terra-01",
		Version:  "v1",
		Repo:     "docker.io/stellarproject",
		OS:       "docker.io/stellarproject/terraos:v10",
		Init:     "/sbin/init",
		Components: []*Component{
			{
				Image:   "docker.io/stellarproject/buildkit:v10",
				Systemd: []string{"buildkit"},
			},
		},
		Userland: "RUN apt install htop",
		SSH: SSH{
			Github: "crosbymichael",
		},
		Netplan: &netplan.Netplan{
			Interfaces: []netplan.Interface{
				{
					Name:      "eth0",
					Addresses: []string{"192.168.1.10"},
					Gateway:   "192.168.1.1",
				},
			},
		},
		ResolvConf: &resolvconf.Conf{
			Nameservers: resolvconf.DefaultNameservers,
		},
	}

	return toml.NewEncoder(os.Stdout).Encode(c)
}
