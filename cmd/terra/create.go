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
	"github.com/urfave/cli"
)

var createCommand = cli.Command{
	Name:  "create",
	Usage: "create a new machine image",
	Flags: []cli.Flag{
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
		cli.StringFlag{
			Name:  "vhost-mount",
			Usage: "vhost containerd mount path",
		},
	},
	Action: func(clix *cli.Context) error {
		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		dest, err := filepath.Abs(".")
		if err != nil {
			return errors.Wrap(err, "get context abs")
		}
		ctx := cmd.CancelContext()
		imageContext := &ImageContext{
			Base:     node.Image.Base,
			Userland: node.Image.Userland,
			Init:     node.Image.Init,
			Hostname: node.Hostname,
		}
		if userland := clix.String("userland"); userland != "" {
			data, err := ioutil.ReadFile(userland)
			if err != nil {
				return errors.Wrapf(err, "read userland file %s", userland)
			}
			imageContext.Userland = string(data)
		}
		if err := node.InstallConfig(dest); err != nil {
			return errors.Wrap(err, "install node configuration to context")
		}

		for _, c := range node.Image.Components {
			imageContext.Imports = append(imageContext.Imports, &cmd.Component{
				Image:   c.Image,
				Systemd: c.Systemd,
			})
		}
		if err := writeDockerfile(dest, imageContext, serverTemplate); err != nil {
			return errors.Wrap(err, "write dockerfile")
		}
		if clix.Bool("dry") {
			fmt.Printf("dumped data to %s\n", dest)
			return nil
		}
		ref := node.Image.Name
		cmd := exec.CommandContext(ctx, "vab", "build",
			"-c", dest,
			"--ref", ref,
			"--push="+strconv.FormatBool(clix.Bool("push")),
			"--no-cache="+strconv.FormatBool(clix.Bool("no-cache")),
			"--http="+strconv.FormatBool(clix.Bool("http")),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = dest

		if err := cmd.Run(); err != nil {
			f, ferr := os.Open(filepath.Join(dest, "Dockerfile"))
			if ferr != nil {
				return errors.Wrap(ferr, "open Dockerfile")
			}
			defer f.Close()
			io.Copy(os.Stdout, f)
			return errors.Wrap(err, "execute build")
		}
		var uid int
		if vhost := clix.String("vhost"); vhost != "" {
			c := &v1.Container{
				ID:         fmt.Sprintf("%s-vhost", node.Hostname),
				Image:      ref,
				Privileged: true,
				UID:        &uid,
				GID:        &uid,
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
			if m := clix.String("vhost-mount"); m != "" {
				c.Mounts = append(c.Mounts, v1.Mount{
					Type:        "bind",
					Source:      m,
					Destination: "/var/lib/containerd",
					Options:     []string{"bind", "rw"},
				})
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

func writeDockerfile(path string, ctx *ImageContext, temp string) error {
	f, err := os.Create(filepath.Join(path, "Dockerfile"))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := render(f, temp, ctx); err != nil {
		return err
	}
	return nil
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

ADD etc/hostname /etc/
ADD etc/hosts /etc/
ADD etc/resolv.conf /etc/
ADD etc/hostname /etc/
ADD etc/netplan/01-netcfg.yaml /etc/netplan/

ADD home/terra/.ssh /home/terra/.ssh

RUN chown -R terra:terra /home/terra

RUN dbus-uuidgen --ensure=/etc/machine-id && dbus-uuidgen --ensure

{{.Userland}}

{{if .Init}}CMD ["{{.Init}}"]{{end}}
`

func cname(c *cmd.Component) string {
	h := md5.New()
	h.Write([]byte(c.Image))
	return "I" + hex.EncodeToString(h.Sum(nil))
}

func cmdargs(args []string) string {
	return strings.Join(args, " ")
}

func imageName(c *cmd.Component) string {
	return c.Image
}

func render(w io.Writer, temp string, ctx *ImageContext) error {
	t, err := template.New("dockerfile").Funcs(template.FuncMap{
		"cname":     cname,
		"imageName": imageName,
		"cmdargs":   cmdargs,
	}).Parse(temp)
	if err != nil {
		return err
	}
	return t.Execute(w, ctx)
}

type ImageContext struct {
	Base       string
	Userland   string
	Imports    []*cmd.Component
	Kernel     string
	Init       string
	Hostname   string
	ResolvConf bool
}
