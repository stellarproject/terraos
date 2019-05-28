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

package config

import (
	"github.com/containerd/typeurl"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1/services"
)

const Version = "v1"

type Container struct {
	ConfigVersion string `toml:"config_version"`

	ID           string       `toml:"id"`
	Image        string       `toml:"image"`
	Resources    *Resources   `toml:"resources"`
	GPUs         *GPUs        `toml:"gpus"`
	Mounts       []Mount      `toml:"mounts"`
	Env          []string     `toml:"env"`
	Args         []string     `toml:"args"`
	UID          *int         `toml:"uid"`
	GID          *int         `toml:"gid"`
	Networks     []*Network   `toml:"networks"`
	Services     []string     `toml:"services"`
	Configs      []ConfigFile `toml:"configs"`
	Readonly     bool         `toml:"readonly"`
	Capabilities []string     `toml:"capabilities"`
	Privileged   bool         `toml:"privileged"`
	Pty          bool         `toml:"pty"`
	MaskedPaths  []string     `toml:"masked_paths"`
}

type Network struct {
	Type   string `toml:"type"`
	Name   string `toml:"name"`
	Master string `toml:"master"`
	Bridge string `toml:"bridge"`
	IPAM   IPAM   `toml:"ipam"`
}

type IPAM struct {
	Type        string `toml:"type"`
	Subnet      string `toml:"subnet"`
	SubnetRange string `toml:"subnet_range"`
	Gateway     string `toml:"gateway"`
}

func (c *Container) Proto() (*v1.Container, error) {
	container := &v1.Container{
		ID:    c.ID,
		Image: c.Image,
		Process: &v1.Process{
			Args: c.Args,
			Env:  c.Env,
			Pty:  c.Pty,
		},
		Readonly: c.Readonly,
		Security: &v1.Security{
			Privileged:   c.Privileged,
			Capabilities: c.Capabilities,
			MaskedPaths:  c.MaskedPaths,
		},
	}
	if len(c.Networks) == 0 {
		return nil, errors.New("no networks provided for container")
	}
	for _, n := range c.Networks {
		switch n.Type {
		case "host":
			any, err := typeurl.MarshalAny(&v1.HostNetwork{})
			if err != nil {
				return nil, errors.Wrap(err, "marshal host network")
			}
			container.Networks = append(container.Networks, any)
		default:
			cni := &v1.CNINetwork{
				Type:   n.Type,
				Name:   n.Name,
				Master: n.Master,
				Bridge: n.Bridge,
			}
			if n.IPAM.Type != "" {
				cni.IPAM = &v1.CNIIPAM{
					Type:        n.IPAM.Type,
					Subnet:      n.IPAM.Subnet,
					SubnetRange: n.IPAM.SubnetRange,
					Gateway:     n.IPAM.Gateway,
				}
			}
			any, err := typeurl.MarshalAny(cni)
			if err != nil {
				return nil, errors.Wrap(err, "marshal cni network")
			}
			container.Networks = append(container.Networks, any)
		}
	}
	for _, m := range c.Mounts {
		container.Mounts = append(container.Mounts, &v1.Mount{
			Type:        m.Type,
			Source:      m.Source,
			Destination: m.Destination,
			Options:     m.Options,
		})
	}
	if c.Resources != nil {
		container.Resources = &v1.Resources{
			Cpus:   c.Resources.CPU,
			Memory: c.Resources.Memory,
			Score:  c.Resources.Score,
			NoFile: c.Resources.NoFile,
		}
	}
	if c.GPUs != nil {
		container.Gpus = &v1.GPUs{
			Devices:      c.GPUs.Devices,
			Capabilities: c.GPUs.Capabilities,
		}
	}
	if c.UID != nil {
		gid := 0
		if c.GID != nil {
			gid = *c.GID
		}
		container.Process.User = &v1.User{
			Uid: uint32(*c.UID),
			Gid: uint32(gid),
		}
	}
	for _, s := range c.Services {
		container.Services = append(container.Services, s)
	}
	for _, v := range c.Configs {
		container.Configs = append(container.Configs, &v1.ConfigFile{
			ID:   v.ID,
			Path: v.Path,
		})
	}
	return container, nil
}

type ConfigFile struct {
	ID   string `toml:"id"`
	Path string `toml:"path"`
}

type Resources struct {
	CPU    float64 `toml:"cpu"`
	Memory int64   `toml:"memory"`
	Score  int64   `toml:"score"`
	NoFile uint64  `toml:"no_file"`
}

type GPUs struct {
	Devices      []int64  `toml:"devices"`
	Capabilities []string `toml:"capabilities"`
}

type Mount struct {
	Type        string   `toml:"type"`
	Source      string   `toml:"source"`
	Destination string   `toml:"destination"`
	Options     []string `toml:"options"`
}
