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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stellarproject/orbit/pkg/resolvconf"
	v1 "github.com/stellarproject/terraos/api/node/v1"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/netplan"
	"github.com/urfave/cli"
)

const hostsTemplate = `127.0.0.1       localhost %s
::1             localhost ip6-localhost ip6-loopback
ff02::1         ip6-allnodes
ff02::2         ip6-allrouters`

var dirs = []string{
	"/var/lib/containerd",
}

// fstab
// resolv.conf
// hosts
// hostname
// ssh
// netplan
// machine id
var configureCommand = cli.Command{
	Name:        "_configure",
	Description: "configure a node's install",
	Hidden:      true,
	Action: func(clix *cli.Context) error {
		ctx := cmd.CancelContext()
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return errors.Wrap(err, "read provision request")
		}
		var request v1.ProvisionRequest
		if err := proto.Unmarshal(data, &request); err != nil {
			return errors.Wrap(err, "unmarshal provision proto")
		}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return errors.Wrapf(err, "mkdir %s", d)
			}
		}
		if err := setupResolvConf(&request); err != nil {
			return errors.Wrap(err, "setup /etc/resolv.conf")
		}
		if err := setupSSH(&request); err != nil {
			return errors.Wrap(err, "setup ssh")
		}
		if err := setupHostname(&request); err != nil {
			return errors.Wrap(err, "setup hostname")
		}
		if err := setupNetplan(&request); err != nil {
			return errors.Wrap(err, "setup netplan")
		}
		if err := setupFstab(&request); err != nil {
			return errors.Wrap(err, "setup fstab")
		}
		if err := setupMachineID(ctx); err != nil {
			return errors.Wrap(err, "setup machine id")
		}
		return nil
	},
}

func setupFstab(r *v1.ProvisionRequest) error {
	// add the fstable entrires first
	var entries []*fstab.Entry
	if r.ClusterFs != "" {
		entries = []*fstab.Entry{
			&fstab.Entry{
				Type:   "9p",
				Device: r.ClusterFs,
				Path:   "/cluster",
				Pass:   2,
				Options: []string{
					"port=564",
					"version=9p2000.L",
					"uname=root",
					"access=user",
					"aname=/cluster",
				},
			},
		}
	}
	for _, v := range r.Node.Volumes {
		entries = append(entries, v.Entries()...)
	}
	f, err := os.Create(fstab.Path)
	if err != nil {
		return errors.Wrap(err, "create fstab file")
	}
	defer f.Close()
	if err := fstab.Write(f, entries); err != nil {
		return errors.Wrap(err, "write fstab")
	}
	return nil
}

func setupMachineID(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "dbus-uuidgen", "--ensure=/etc/machine-id")
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	cmd = exec.CommandContext(ctx, "dbus-uuidgen", "--ensure")
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}

func setupResolvConf(r *v1.ProvisionRequest) error {
	resolv := &resolvconf.Conf{
		Nameservers: r.Nameservers,
	}
	if resolv.Nameservers == nil {
		resolv.Nameservers = resolvconf.DefaultNameservers
	}
	f, err := os.Create(resolvconf.DefaultPath)
	if err != nil {
		return errors.Wrap(err, "create resolv.conf file")
	}
	defer f.Close()

	if err := resolv.Write(f); err != nil {
		return errors.Wrap(err, "write resolv.conf")
	}
	return nil
}

func setupHostname(r *v1.ProvisionRequest) error {
	f, err := os.Create("/etc/hostname")
	if err != nil {
		return errors.Wrap(err, "create hostname file")
	}
	_, err = f.WriteString(r.Node.Hostname)
	f.Close()
	if err != nil {
		return errors.Wrap(err, "write hostname contents")
	}

	if f, err = os.Create("/etc/hosts"); err != nil {
		return errors.Wrap(err, "create hosts file")
	}
	_, err = fmt.Fprintf(f, hostsTemplate, r.Node.Hostname)
	f.Close()
	if err != nil {
		return errors.Wrap(err, "write hosts contents")
	}
	return nil
}

func setupSSH(r *v1.ProvisionRequest) error {
	if err := os.MkdirAll("/home/terra/.ssh", 0711); err != nil {
		return errors.Wrap(err, "create .ssh dir")
	}
	f, err := os.Create("/home/terra/.ssh/authorized_keys")
	if err != nil {
		return errors.Wrap(err, "create ssh key file")
	}
	defer f.Close()

	for _, key := range r.SshKeys {
		if _, err := fmt.Fprintf(f, "%s\n", key); err != nil {
			return errors.Wrap(err, "write ssh key")
		}
	}
	return nil
}

func setupNetplan(r *v1.ProvisionRequest) error {
	n := &netplan.Netplan{}
	for _, nic := range r.Node.Nics {
		n.Interfaces = append(n.Interfaces, &netplan.Interface{
			Name:      nic.Name,
			Addresses: nic.Addresses,
			Gateway:   r.Gateway,
		})
	}
	p := filepath.Join("/etc/netplan", netplan.DefaultFilename)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return errors.Wrap(err, "create netplan dir")
	}
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
