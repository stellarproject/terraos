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
	v1 "github.com/stellarproject/terraos/api/v1/types"
)

type Node struct {
	Hostname string      `toml:"hostname"`
	Mac      string      `toml:"mac"`
	Image    string      `toml:"image"`
	Groups   []DiskGroup `toml:"groups"`
}

type Disk struct {
	Device string `toml:"device"`
	Size   int64  `toml:"size"`
}

type DiskGroup struct {
	Label      string      `toml:"label"`
	Type       string      `toml:"type"`
	Subvolumes []Subvolume `toml:"subvolumes"`
	Stage      string      `toml:"stage"`
	Disks      []Disk      `toml:"disk"`
	Mbr        bool        `toml:"mbr"`
}

type Subvolume struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
	COW  bool   `toml:"cow"`
}

func (n *Node) ToProto() *v1.Node {
	p := &v1.Node{
		Hostname: n.Hostname,
		Mac:      n.Mac,
		Image:    n.Image,
	}
	for _, g := range n.Groups {
		var (
			stage v1.Stage
			t     v1.DiskGroupType
		)
		switch g.Stage {
		case "stage0":
			stage = v1.Stage0
		case "stage1":
			stage = v1.Stage1
		}
		switch g.Type {
		case "single":
			t = v1.Single
		case "raid0":
			t = v1.RAID0
		case "raid5":
			t = v1.RAID5
		case "raid10":
			t = v1.RAID10
		}
		p.DiskGroups = append(p.DiskGroups, &v1.DiskGroup{
			GroupType:  t,
			Stage:      stage,
			Label:      g.Label,
			Subvolumes: subvolumes(g.Subvolumes),
			Disks:      disks(g.Disks),
			Mbr:        g.Mbr,
		})
	}

	return p
}

func disks(disks []Disk) (out []*v1.Disk) {
	for _, d := range disks {
		out = append(out, &v1.Disk{
			Device: d.Device,
			FsSize: d.Size,
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
		Mac:      "66:xx:ss:bb:f1:b1",
		Image:    "docker.io/stellarproject/example:v4",
		Groups: []DiskGroup{
			{
				Stage: "stage0",
				Type:  "single",
				Disks: []Disk{
					{
						Device: "/dev/sda1",
					},
				},
			},
			{
				Label: "os",
				Stage: "stage1",
				Type:  "raid0",
				Disks: []Disk{
					{
						Device: "/dev/sda2",
					},
					{
						Device: "/dev/sdb",
						Size:   512,
					},
				},
				Subvolumes: []Subvolume{
					{
						Name: "tftp",
						Path: "/tftp",
					},
				},
			},
		},
	}
	return toml.NewEncoder(os.Stdout).Encode(c)
}
