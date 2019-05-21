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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
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

		if err := setupHostname(abs, config.Hostname, &paths); err != nil {
			return errors.Wrap(err, "setup hostname")
		}
		if err := setupSSH(abs, config.SSH, &paths); err != nil {
			return errors.Wrap(err, "setup ssh keys")
		}
		if err := setupNetplan(abs, config.Netplan, &paths); err != nil {
			return errors.Wrap(err, "setup netplan")
		}
		if err := setupResolvConf(abs, config.ResolvConf, &paths); err != nil {
			return errors.Wrap(err, "setup resolv.conf")
		}

		if clix.Bool("dry") {
			return nil
		}
		cmd := exec.CommandContext(ctx, "vab", "build",
			"-c", abs,
			"--ref", fmt.Sprintf("%s/%s:%s", config.Repo, config.Hostname, config.Version),
			"--push="+strconv.FormatBool(clix.Bool("push")),
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
{{if ne $v.Name "kernel"}}
COPY --from={{cname $v}} / /
{{range $s := $v.Systemd}}
RUN systemctl enable {{$s}}
{{end}}
{{end}}
{{end}}

ADD hostname /etc/hostname
ADD hosts /etc/hosts
ADD resolv.conf /etc/resolv.conf
ADD 01-netcfg.yaml /etc/netplan/

RUN mkdir -p /home/terra/.ssh /var/log /var/lib/containerd
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

func cname(c Component) string {
	h := md5.New()
	h.Write([]byte(c.Image))
	return hex.EncodeToString(h.Sum(nil))
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

func setupHostname(path, hostname string, paths *[]string) error {
	if hostname == "" {
		return errors.New("cannot use empty hostname")
	}
	f, err := os.Create(filepath.Join(path, "hostname"))
	if err != nil {
		return errors.Wrap(err, "create hostname file")
	}
	*paths = append(*paths, f.Name())
	_, err = f.WriteString(hostname)
	f.Close()

	if err != nil {
		return errors.Wrap(err, "write hostname contents")
	}
	if f, err = os.Create(filepath.Join(path, "hosts")); err != nil {
		return errors.Wrap(err, "create hosts file")
	}
	*paths = append(*paths, f.Name())
	_, err = fmt.Fprintf(f, hostsTemplate, hostname)
	f.Close()
	if err != nil {
		return errors.Wrap(err, "write hosts contents")
	}
	return nil
}

func setupSSH(path string, ssh SSH, paths *[]string) error {
	p := filepath.Join(path, "keys")
	*paths = append(*paths, p)
	if ssh.Github != "" {
		r, err := http.Get(fmt.Sprintf("https://github.com/%s.keys", ssh.Github))
		if err != nil {
			return errors.Wrap(err, "fetch ssh keys")
		}
		defer r.Body.Close()
		f, err := os.Create(p)
		if err != nil {
			return errors.Wrap(err, "create ssh key file")
		}
		defer f.Close()
		if _, err := io.Copy(f, r.Body); err != nil {
			return errors.Wrap(err, "copy ssh key contents")
		}
	} else {
		f, err := os.Create(p)
		if err != nil {
			return errors.Wrap(err, "create empty ssh key file")
		}
		f.Close()
	}
	return nil
}

func setupResolvConf(path string, resolv *resolvconf.Conf, paths *[]string) error {
	if resolv == nil {
		return nil
	}
	p := filepath.Join(path, "resolv.conf")
	// don't create a resolv conf if it already exists in the context
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	if resolv.Nameservers == nil {
		resolv.Nameservers = resolvconf.DefaultNameservers
	}
	f, err := os.Create(p)
	if err != nil {
		return errors.Wrap(err, "create resolv.conf file")
	}
	defer f.Close()
	*paths = append(*paths, p)

	if err := resolv.Write(f); err != nil {
		return errors.Wrap(err, "write resolv.conf")
	}
	return nil
}

func setupNetplan(path string, n *netplan.Netplan, paths *[]string) error {
	if n == nil {
		return nil
	}
	p := filepath.Join(path, netplan.DefaultFilename)
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	*paths = append(*paths, p)
	f, err := os.Create(p)
	if err != nil {
		return errors.Wrap(err, "create netplan file")
	}
	defer f.Close()
	if err := n.Write(f); err != nil {
		return errors.Wrap(err, "write netplan contents")
	}
	return nil
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
