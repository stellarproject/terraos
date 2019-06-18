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
	Nics     []NIC    `toml:"nic"`
	Volumes  []Volume `toml:"volume"`
	GPUs     []GPU    `toml:"gpus"`
	CPUs     []CPU    `toml:"cpus"`
	Memory   uint32   `toml:"memory"`
	Domain   string   `toml:"domain"`
}

type NIC struct {
	Mac       string   `toml:"mac"`
	Addresses []string `toml:"addresses"`
	Speed     uint32   `toml:"speed"`
}

type Disk struct {
	Device string `toml:"device"`
}

type Volume struct {
	Label      string      `toml:"label"`
	Type       string      `toml:"type"`
	Subvolumes []Subvolume `toml:"subvolumes"`
	Disks      []Disk      `toml:"disk"`
	FSType     string      `toml:"fs_type"`
	Mbr        bool        `toml:"mbr"`
	Path       string      `toml:"path"`
	Size       int64       `toml:"size"`
}

type Subvolume struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
	COW  bool   `toml:"cow"`
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
		Hostname: n.Hostname,
		Domain:   n.Domain,
		Memory:   n.Memory,
		Labels:   n.Labels,
	}
	for _, g := range n.Volumes {
		t := v1.Single
		switch g.Type {
		case "single":
			t = v1.Single
		case "raid0":
			t = v1.RAID0
		case "raid5":
			t = v1.RAID5
		case "raid10":
			t = v1.RAID10
		case "iscsi":
			t = v1.ISCSIVolume
		}
		p.Volumes = append(p.Volumes, &v1.Volume{
			Path:       g.Path,
			Type:       t,
			Label:      g.Label,
			Subvolumes: subvolumes(g.Subvolumes),
			Disks:      disks(g.Disks),
			Boot:       g.Mbr,
			FsSize:     g.Size,
		})
	}
	for _, nic := range n.Nics {
		p.Nics = append(p.Nics, &v1.NIC{
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

func disks(disks []Disk) (out []*v1.Disk) {
	for _, d := range disks {
		out = append(out, &v1.Disk{
			Device: d.Device,
		})
	}
	return out
}

func subvolumes(subvolumes []Subvolume) (out []*v1.Subvolume) {
	for _, s := range subvolumes {
		out = append(out, &v1.Subvolume{
			Name: s.Name,
			Path: s.Path,
			Cow:  s.COW,
		})
	}
	return out
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
		Nics: []NIC{
			{
				Mac:       "66:xx:ss:bb:f1:b1",
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
				Type:   "single",
				Path:   "/",
				Label:  "os",
				FSType: "ext4",
				Disks: []Disk{
					{
						Device: "/dev/sda1",
					},
				},
			},
			{
				Label:  "data",
				Type:   "raid10",
				FSType: "btrfs",
				Disks: []Disk{
					{
						Device: "/dev/sda2",
					},
					{
						Device: "/dev/sdb",
					},
				},
				Subvolumes: []Subvolume{
					{
						Name: "tftp",
						Path: "/tftp",
					},
				},
			},
			{
				Type:   "iscsi",
				Label:  "containerd",
				Path:   "/var/lib/containerd",
				FSType: "ext4",
				Size:   64000,
			},
		},
	}
	return toml.NewEncoder(os.Stdout).Encode(c)
}
