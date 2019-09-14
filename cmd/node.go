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

package cmd

import (
	"net/http"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/types/v1"
)

type Node struct {
	Hostname string   `toml:"hostname"`
	Labels   []string `toml:"labels"`
	Volumes  []Volume `toml:"volumes"`
	GPUs     []GPU    `toml:"gpus"`
	CPUs     float64  `toml:"cpus"`
	Memory   uint32   `toml:"memory"`
	Domain   string   `toml:"domain"`
	Image    Image    `toml:"image"`
	PXE      PXE      `toml:"pxe"`
	Network  Network  `toml:"network"`
}

type Network struct {
	Interfaces  string      `toml:"interfaces"`
	Nameservers []string    `toml:"nameservers"`
	Gateway     string      `toml:"gateway"`
	PXE         *PXENetwork `toml:"pxe"`
}

type PXENetwork struct {
	Mac       string   `toml:"mac"`
	Address   string   `toml:"address"`
	Bond      []string `toml:"bond"`
	Interface string   `toml:"interface"`
}

type Image struct {
	Name       string       `toml:"name"`
	Base       string       `toml:"base"`
	Init       string       `toml:"init"`
	Components []*Component `toml:"components"`
	Userland   string       `toml:"userland"`
	SSH        SSH          `toml:"ssh"`
	Packages   []string     `toml:"packages"`
	Services   []string     `toml:"services"`
}

type SSH struct {
	Github string   `toml:"github"`
	Keys   []string `toml:"keys"`
}

type Component struct {
	Image string   `toml:"image"`
	Init  []string `toml:"init"`
}

type Disk struct {
	Device string `toml:"device"`
}

type Volume struct {
	Label     string `toml:"label"`
	Type      string `toml:"type"`
	Path      string `toml:"path"`
	Boot      bool   `toml:"boot"`
	TargetIQN string `toml:"target_iqn"`
}

type GPU struct {
	Model        string   `toml:"model"`
	Cores        uint32   `toml:"cores"`
	Memory       uint32   `toml:"memory"`
	Capabilities []string `toml:"capabilities"`
}

type PXE struct {
	Target string `toml:"target"`
}

func (n *Node) ToProto() *v1.Node {
	p := &v1.Node{
		Hostname: n.Hostname,
		Domain:   n.Domain,
		Memory:   n.Memory,
		Labels:   n.Labels,
		Cpus:     n.CPUs,
		Network: &v1.Network{
			Gateway:     n.Network.Gateway,
			Nameservers: n.Network.Nameservers,
		},
		Image: &v1.Image{
			Name:     n.Image.Name,
			Base:     n.Image.Base,
			Init:     n.Image.Init,
			Userland: n.Image.Userland,
			Packages: n.Image.Packages,
			Services: n.Image.Services,
			Ssh: &v1.SSH{
				Github: n.Image.SSH.Github,
				Keys:   n.Image.SSH.Keys,
			},
		},
		Pxe: &v1.PXE{
			IscsiTarget: n.PXE.Target,
		},
	}
	if n.Network.PXE != nil {
		p.Network.PxeNetwork = &v1.PXENetwork{
			Mac:       n.Network.PXE.Mac,
			Address:   n.Network.PXE.Address,
			Bond:      n.Network.PXE.Bond,
			Interface: n.Network.PXE.Interface,
		}
	}
	for _, c := range n.Image.Components {
		p.Image.Components = append(p.Image.Components, &v1.Component{
			Image:   c.Image,
			Systemd: c.Init,
		})
	}
	for _, g := range n.Volumes {
		p.Volumes = append(p.Volumes, &v1.Volume{
			Path:      g.Path,
			Type:      g.Type,
			Label:     g.Label,
			Boot:      g.Boot,
			TargetIqn: g.TargetIQN,
		})
	}
	for _, g := range n.GPUs {
		p.Gpus = append(p.Gpus, &v1.GPU{
			Model:        g.Model,
			Cores:        g.Cores,
			Memory:       g.Memory,
			Capabilities: g.Capabilities,
		})
	}
	return p
}

func LoadNode(path string) (*v1.Node, error) {
	var node Node

	uri, err := url.Parse(path)
	if err != nil {
		return nil, errors.Wrap(err, "parse path")
	}
	switch uri.Scheme {
	case "http", "https":
		r, err := http.Get(path)
		if err != nil {
			return nil, errors.Wrap(err, "http get node")
		}
		defer r.Body.Close()
		if _, err := toml.DecodeReader(r.Body, &node); err != nil {
			return nil, errors.Wrap(err, "load node file")
		}
	default:
		if _, err := toml.DecodeFile(path, &node); err != nil {
			return nil, errors.Wrap(err, "load node file")
		}
	}
	return node.ToProto(), nil
}

func DumpNodeConfig() error {
	c := &Node{
		Hostname: "terra-01",
		Domain:   "home",
		Labels:   []string{"controller", "plex"},
		Memory:   4096,
		Network: Network{
			Gateway:     "192.168.1.1",
			Nameservers: []string{"8.8.8.8", "8.8.4.4"},
		},
		Image: Image{
			Name:     "docker.io/stellarproject/example:9",
			Base:     "docker.io/stellarproject/terraos:v13",
			Init:     "/sbin/init",
			Userland: "RUN apt update",
			SSH: SSH{
				Github: "crosbymichael",
			},
			Components: []*Component{
				{
					Image: "docker.io/stellarproject/diod:v13",
					Init:  []string{"diod"},
				},
			},
		},
		GPUs: []GPU{
			{
				Model:        "Geforce Titian X",
				Cores:        6400,
				Memory:       12000,
				Capabilities: []string{"compute", "video"},
			},
		},
		CPUs: 4,
		Volumes: []Volume{
			{
				Path:  "/",
				Label: "os",
				Type:  "btrfs",
			},
			{
				Label: "ctd",
				Type:  "btrfs",
				Path:  "/var/lib/containerd",
			},
		},
	}
	return toml.NewEncoder(os.Stdout).Encode(c)
}
