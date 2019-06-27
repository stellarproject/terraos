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
	Hostname    string   `toml:"hostname"`
	Labels      []string `toml:"labels"`
	Nics        []NIC    `toml:"nic"`
	Volumes     []Volume `toml:"volumes"`
	GPUs        []GPU    `toml:"gpus"`
	CPUs        []CPU    `toml:"cpus"`
	Memory      uint32   `toml:"memory"`
	Domain      string   `toml:"domain"`
	Image       Image    `toml:"image"`
	Gateway     string   `toml:"gateway"`
	Nameservers []string `toml:"nameservers"`
	ClusterFS   string   `toml:"cluster_fs"`
}

type Image struct {
	Name       string       `toml:"name"`
	Base       string       `toml:"base"`
	Init       string       `toml:"init"`
	Components []*Component `toml:"components"`
	Userland   string       `toml:"userland"`
	SSH        SSH          `toml:"ssh"`
}

type SSH struct {
	Github string   `toml:"github"`
	Keys   []string `toml:"keys"`
}

type Component struct {
	Image   string   `toml:"image"`
	Systemd []string `toml:"systemd"`
}

type NIC struct {
	Mac       string   `toml:"mac"`
	Addresses []string `toml:"addresses"`
	Speed     uint32   `toml:"speed"`
	Name      string   `toml:"name"`
}

type Disk struct {
	Device string `toml:"device"`
}

type Volume struct {
	Label string `toml:"label"`
	Type  string `toml:"type"`
	Path  string `toml:"path"`
	Size  int64  `toml:"size"`
}

type CPU struct {
	Ghz float64 `toml:"ghz"`
}

type GPU struct {
	Model        string   `toml:"model"`
	Cores        uint32   `toml:"cores"`
	Memory       uint32   `toml:"memory"`
	Capabilities []string `toml:"capabilities"`
}

func (n *Node) ToProto() *v1.Node {
	p := &v1.Node{
		Hostname:    n.Hostname,
		Domain:      n.Domain,
		Memory:      n.Memory,
		Labels:      n.Labels,
		Gateway:     n.Gateway,
		Nameservers: n.Nameservers,
		ClusterFs:   n.ClusterFS,
		Image: &v1.Image{
			Name:     n.Image.Name,
			Base:     n.Image.Base,
			Init:     n.Image.Init,
			Userland: n.Image.Userland,
			Ssh: &v1.SSH{
				Github: n.Image.SSH.Github,
				Keys:   n.Image.SSH.Keys,
			},
		},
	}
	for _, c := range n.Image.Components {
		p.Image.Components = append(p.Image.Components, &v1.Component{
			Image:   c.Image,
			Systemd: c.Systemd,
		})
	}
	for _, g := range n.Volumes {
		p.Volumes = append(p.Volumes, &v1.Volume{
			Path:   g.Path,
			Type:   g.Type,
			Label:  g.Label,
			FsSize: g.Size,
		})
	}
	for _, nic := range n.Nics {
		p.Nics = append(p.Nics, &v1.NIC{
			Name:      nic.Name,
			Mac:       nic.Mac,
			Addresses: nic.Addresses,
			Speed:     nic.Speed,
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
	for _, c := range n.CPUs {
		p.Cpus = append(p.Cpus, &v1.CPU{
			Ghz: c.Ghz,
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
		Hostname:    "terra-01",
		Gateway:     "192.168.1.1",
		Nameservers: []string{"8.8.8.8", "8.8.4.4"},
		ClusterFS:   "192.168.1.10",
		Domain:      "home",
		Labels:      []string{"controller", "plex"},
		Memory:      4096,
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
					Image:   "docker.io/stellarproject/diod:v13",
					Systemd: []string{"diod"},
				},
			},
		},
		Nics: []NIC{
			{
				Name:      "eth0",
				Mac:       "xx:xx:xx:xx:xx:xx",
				Addresses: []string{"192.168.0.10"},
				Speed:     1000,
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
		CPUs: []CPU{
			{
				Ghz: 3.4,
			},
			{
				Ghz: 3.4,
			},
			{
				Ghz: 3.4,
			},
			{
				Ghz: 3.4,
			},
		},
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
