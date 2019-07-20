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

package v1

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/stellarproject/terraos/pkg/netplan"
	"github.com/stellarproject/terraos/pkg/resolvconf"
	"golang.org/x/sys/unix"
)

const (
	year   = 2019
	iqnFmt = "iqn.%d.%s:%s"
)
const hostsTemplate = `127.0.0.1       localhost %s
::1             localhost ip6-localhost ip6-loopback
ff02::1         ip6-allnodes
ff02::2         ip6-allrouters`

func (n *Node) IQN() string {
	return fmt.Sprintf(iqnFmt, year, n.Domain, n.Hostname)
}

func (v *Volume) IsISCSI() bool {
	return v.TargetIqn != ""
}

func (v *Volume) Format(device string) error {
	return mkfs.Mkfs(v.Type, v.Label, device)
}

func (v *Volume) MountLabel() string {
	return fmt.Sprintf("LABEL=%s", v.Label)
}

func (v *Volume) Mount(device, dest string) (func() error, error) {
	p := filepath.Join(dest, v.Path)
	if err := os.MkdirAll(p, 0755); err != nil {
		return nil, errors.Wrapf(err, "mkdir %s", p)
	}
	if err := unix.Mount(device, p, v.Type, 0, ""); err != nil {
		return nil, errors.Wrapf(err, "mount %s to %s", v.Label, p)
	}
	return func() error {
		return unix.Unmount(p, 0)
	}, nil
}

func (v *Volume) Entries() []*fstab.Entry {
	return []*fstab.Entry{
		&fstab.Entry{
			Type:   v.Type,
			Pass:   2,
			Device: fmt.Sprintf("LABEL=%s", v.Label),
			Path:   v.Path,
		},
	}
}

func (i *Node) InstallConfig(dest string) error {
	if err := i.setupFstab(dest); err != nil {
		return err
	}
	if err := i.setupHostname(dest); err != nil {
		return err
	}
	if err := i.setupNetplan(dest); err != nil {
		return err
	}
	if err := i.setupResolvConf(dest); err != nil {
		return err
	}
	if err := i.setupSSH(dest); err != nil {
		return err
	}
	return nil
}

func (i *Node) setupNetplan(dest string) error {
	n := &netplan.Netplan{}
	for _, nic := range i.Nics {
		gw := i.Gateway
		if len(nic.Addresses) == 0 {
			gw = ""
		}
		n.Interfaces = append(n.Interfaces, netplan.Interface{
			Name:      nic.Name,
			Addresses: nic.Addresses,
			Gateway:   gw,
		})
	}
	p := filepath.Join(dest, "/etc/netplan", netplan.DefaultFilename)
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

func (i *Node) setupFstab(dest string) error {
	var entries []*fstab.Entry

	if i.ClusterFs != "" {
		entries = []*fstab.Entry{
			&fstab.Entry{
				Type:   "9p",
				Device: i.ClusterFs,
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
	for _, v := range i.Volumes {
		entries = append(entries, v.Entries()...)
	}
	path := filepath.Join(dest, fstab.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "create base path")
	}

	f, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "create fstab file")
	}
	defer f.Close()

	if err := fstab.Write(f, entries); err != nil {
		return errors.Wrap(err, "write fstab")
	}
	return nil
}

func (i *Node) setupResolvConf(dest string) error {
	resolv := &resolvconf.Conf{
		Nameservers: i.Nameservers,
	}
	if len(resolv.Nameservers) == 0 {
		resolv.Nameservers = []string{
			i.Gateway,
		}
	}
	path := filepath.Join(dest, resolvconf.DefaultPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "create base path")
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := resolv.Write(f); err != nil {
		return errors.Wrap(err, "write resolv.conf")
	}
	return nil
}

func (i *Node) setupHostname(dest string) error {
	path := filepath.Join(dest, "/etc/hostname")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "create base path")
	}
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "create hostname file")
	}
	_, err = f.WriteString(i.Hostname)
	f.Close()
	if err != nil {
		return errors.Wrap(err, "write hostname contents")
	}
	if f, err = os.Create(filepath.Join(dest, "/etc/hosts")); err != nil {
		return errors.Wrap(err, "create hosts file")
	}
	_, err = fmt.Fprintf(f, hostsTemplate, i.Hostname)
	f.Close()

	if err != nil {
		return errors.Wrap(err, "write hosts contents")
	}
	return nil
}

func (i *Node) setupSSH(dest string) error {
	if i.Image.Ssh == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(dest, "/home/terra/.ssh"), 0755); err != nil {
		return errors.Wrap(err, "create .ssh dir")
	}
	f, err := os.Create(filepath.Join(dest, "/home/terra/.ssh/authorized_keys"))
	if err != nil {
		return errors.Wrap(err, "create ssh key file")
	}
	defer f.Close()

	if err := f.Chmod(0644); err != nil {
		return errors.Wrap(err, "chmod ssh files")
	}

	if i.Image.Ssh.Github != "" {
		r, err := http.Get(i.Image.Ssh.Github)
		if err != nil {
			return errors.Wrap(err, "fetch github keys")
		}
		defer r.Body.Close()
		if _, err := io.Copy(f, r.Body); err != nil {
			return errors.Wrap(err, "write github ssh keys")
		}
		f.WriteString("\n")
	}

	for _, key := range i.Image.Ssh.Keys {
		if _, err := fmt.Fprintf(f, "%s\n", key); err != nil {
			return errors.Wrap(err, "write ssh key")
		}
	}
	return nil
}
