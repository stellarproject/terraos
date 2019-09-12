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
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/mkfs"
	"github.com/stellarproject/terraos/pkg/resolvconf"
	"golang.org/x/sys/unix"
)

var (
	ErrNotISCSIVolume = errors.New("not an iscsi volume")
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

func (v *Volume) Login(ctx context.Context, portal string) (string, error) {
	if !v.IsISCSI() {
		return "", ErrNotISCSIVolume
	}
	out, err := exec.CommandContext(ctx, "iscsiadm",
		"--mode", "node",
		"--targetname", v.TargetIqn,
		"--portal", portal,
		"--login").CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "%s", out)
	}
	path := filepath.Join("/dev/disk/by-path", iscsiDeviceByPath(portal, v.TargetIqn))
	dev, err := os.Readlink(path)
	if err != nil {
		v.Logout(ctx, portal)
		return "", errors.Wrap(err, "get iscsi disk by path")
	}
	return dev, nil
}

func iscsiDeviceByPath(portal, iqn string) string {
	// example: ip-10.0.10.10:3260-iscsi-iqn.2019.com.crosbymichael.core:reactor-lun-0-part1
	return fmt.Sprintf("ip-%s:3260-iscsi-%s-lun-0-part1", portal, iqn)
}

func (v *Volume) Logout(ctx context.Context, portal string) error {
	if !v.IsISCSI() {
		return ErrNotISCSIVolume
	}
	out, err := exec.CommandContext(ctx, "iscsiadm",
		"--mode", "node",
		"--targetname", v.TargetIqn,
		"--portal", portal,
		"--logout").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
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
		return errors.Wrap(err, "setup fstab")
	}
	if err := i.setupHostname(dest); err != nil {
		return errors.Wrap(err, "setup hostname")
	}
	if err := i.setupNetworking(dest); err != nil {
		return errors.Wrap(err, "setup networking")
	}
	if err := i.setupResolvConf(dest); err != nil {
		return errors.Wrap(err, "setup resolv.conf")
	}
	if err := i.setupSSH(dest); err != nil {
		return errors.Wrap(err, "setup ssh")
	}
	return nil
}

const loInterfaces = `auto lo
iface lo inet loopback

`

func (i *Node) setupNetworking(dest string) error {
	path := filepath.Join(dest, "etc/networking/interfaces")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "create base path")
	}
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "create interfaces file")
	}
	defer f.Close()

	if _, err := f.WriteString(loInterfaces); err != nil {
		return err
	}
	if i.Network.Interfaces != "" {
		if _, err := f.WriteString(i.Network.Interfaces); err != nil {
			return err
		}
	}
	return nil
}

func (i *Node) setupFstab(dest string) error {
	var entries []*fstab.Entry

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
		Nameservers: i.Network.Nameservers,
	}
	if len(resolv.Nameservers) == 0 {
		resolv.Nameservers = []string{
			i.Network.Gateway,
		}
	}
	path := filepath.Join(dest, resolvconf.DefaultPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "create base path")
	}

	// remove the existing resolv.conf incase it is a symlink
	os.Remove(path)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if len(resolv.Nameservers) == 0 {
		return nil
	}

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

const (
	kvFmt  = "%s-%s"
	kernel = "vmlinuz"
	initrd = "initrd.img"

	pxeTemplate = `DEFAULT pxe

LABEL pxe
  KERNEL {{.Kernel}}
  INITRD {{.Initrd}}
  APPEND {{.Append}}
`
)

type pxeConfig struct {
	Kernel string
	Initrd string
	Append string
}

func PXEFilename(i *Node) string {
	return fmt.Sprintf("01-%s", strings.Replace(i.Network.PxeNetwork.Mac, ":", "-", -1))
}

// PXEConfig writes the pxe config to the writer for the node
func (i *Node) PXEConfig(w io.Writer) error {
	pn := i.Network.PxeNetwork
	ip := pn.Address
	if ip == "" {
		ip = "dhcp"
		if len(pn.Bond) > 0 {
			ip = "none"
		}
	}
	c := &pxeConfig{
		Kernel: fmt.Sprintf(kvFmt, kernel, i.Pxe.Version),
		Initrd: fmt.Sprintf(kvFmt, initrd, i.Pxe.Version),
	}
	args := []string{
		"ip=" + ip,
		"boot=pxe",
		"root=LABEL=os",
	}
	if len(pn.Bond) > 0 {
		args = append(args, fmt.Sprintf("bondslaves=%s", strings.Join(pn.Bond, ",")))
	}
	var target string
	for _, v := range i.Volumes {
		if v.Label == "os" {
			target = v.TargetIqn
			break
		}
	}
	if target == "" {
		return errors.New("no target iqn specified")
	}

	args = append(args,
		fmt.Sprintf("ISCSI_INITIATOR=%s", i.IQN()),
		fmt.Sprintf("ISCSI_TARGET_NAME=%s", target),
		fmt.Sprintf("ISCSI_TARGET_IP=%s", i.Pxe.IscsiTarget),
	)
	c.Append = strings.Join(args, " ")

	t, err := template.New("pxe").Parse(pxeTemplate)
	if err != nil {
		return errors.Wrap(err, "create pxe template")
	}
	return t.Execute(w, c)
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
