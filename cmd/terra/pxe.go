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
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

const (
	configDir = "pxelinux.cfg"
)

var pxeCommand = cli.Command{
	Name:  "pxe",
	Usage: "manage the pxe setup for terra",
	Subcommands: []cli.Command{
		pxeInstallCommand,
		pxeConfigCommand,
	},
}

var pxeInstallCommand = cli.Command{
	Name:      "install",
	Usage:     "install a new pxe image to a directory",
	ArgsUsage: "[image]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "tftp,t",
			Usage: "tftp location",
			Value: "/tftp",
		},
		cli.BoolFlag{
			Name:  "default",
			Usage: "set this pxe install as the default",
		},
		cli.StringFlag{
			Name:  "version",
			Usage: "terraos version",
			Value: version.Version,
		},
	},
	Action: func(clix *cli.Context) error {
		ctx := cmd.CancelContext()
		i := getPXEImage(clix, clix.String("version"))
		store, err := getStore()
		if err != nil {
			return errors.Wrap(err, "get content store")
		}
		img, err := image.Fetch(ctx, clix.GlobalBool("http"), store, i)
		if err != nil {
			return errors.Wrapf(err, "fetch %s", i)
		}
		path, err := ioutil.TempDir("", "terra-pxe-install")
		if err != nil {
			return errors.Wrap(err, "create tmp pxe dir")
		}
		defer os.RemoveAll(path)

		if err := image.Unpack(ctx, store, img, path); err != nil {
			return errors.Wrap(err, "unpack pxe image")
		}
		var (
			source = filepath.Join(path, "tftp") + "/"
			target = clix.String("tftp") + "/"
		)
		if clix.Bool("default") {
			if err := syncDir(ctx, source, target); err != nil {
				return errors.Wrap(err, "sync tftp dir")
			}
		} else {
			if err := copyKernel(source, getVersion(i), target); err != nil {
				return errors.Wrap(err, "copy kernel")
			}
		}
		return nil
	},
}

var pxeConfigCommand = cli.Command{
	Name:      "config",
	Usage:     "configure a node's pxe config",
	ArgsUsage: "[hostname]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "tftp,t",
			Usage: "tftp location",
			Value: "/tftp",
		},
		cli.StringFlag{
			Name:  "ip",
			Usage: "node ip and subnet (192.168.1.5/24)",
		},
		cli.StringFlag{
			Name:  "gateway,g",
			Usage: "node gateway",
		},
		cli.StringFlag{
			Name:  "mac",
			Usage: "node mac address",
		},
		cli.StringFlag{
			Name:  "target-ip",
			Usage: "ISCSI target ip",
		},
		cli.StringFlag{
			Name:  "target-iqn,iqn",
			Usage: "ISCSI target iqn",
		},
		cli.StringFlag{
			Name:  "iqn-base,base",
			Usage: "ISCSI iqn base",
			Value: "terra",
		},
		cli.StringSliceFlag{
			Name:  "nameserver,n",
			Usage: "dns nameserver",
			Value: &cli.StringSlice{},
		},
		cli.StringSliceFlag{
			Name:  "interface,i",
			Usage: "network interface name",
			Value: &cli.StringSlice{},
		},
	},
	Action: func(clix *cli.Context) error {
		hostname := clix.Args().First()
		if hostname == "" {
			return errors.New("hostname must be provided")
		}
		p := &pxe{
			hostname:    hostname,
			mac:         clix.String("mac"),
			network:     clix.String("ip"),
			gateway:     clix.String("gateway"),
			nameservers: clix.StringSlice("nameserver"),
			interfaces:  clix.StringSlice("interface"),
			iscsiIP:     clix.String("target-ip"),
			base:        clix.String("iqn-base"),
			target:      clix.String("target-iqn"),
		}

		path := filepath.Join(clix.String("tftp"), configDir, pxeFilename(p.mac))
		f, err := os.Create(path)
		if err != nil {
			return errors.Wrapf(err, "create pxe config file %s", path)
		}
		defer f.Close()
		if err := p.Write(f, version.Version); err != nil {
			return errors.Wrap(err, "write pxe configuration")
		}
		return nil
	},
}

type pxe struct {
	hostname    string
	mac         string
	network     string
	gateway     string
	nameservers []string
	interfaces  []string
	iscsiIP     string
	target      string
	base        string
}

func (p *pxe) Write(w io.Writer, version string) error {
	if version == "" {
		return errors.New("no version specifeid")
	}
	c := &pxeConfig{
		Kernel: fmt.Sprintf(kvFmt, kernel, version),
		Initrd: fmt.Sprintf(kvFmt, initrd, version),
	}
	args := []string{
		"ip=none",
		"boot=terra",
		fmt.Sprintf("hostname=%s", p.hostname),
		fmt.Sprintf("network=%s", p.network),
		fmt.Sprintf("gateway=%s", p.gateway),
		fmt.Sprintf("nameservers=%s", strings.Join(p.nameservers, ",")),
	}
	if len(p.interfaces) > 1 {
		args = append(args, fmt.Sprintf("bondslaves=%s", strings.Join(p.interfaces, ",")))
	} else {
		args = append(args, fmt.Sprintf("iface=%s", p.interfaces[0]))
	}

	args = append(args,
		fmt.Sprintf("ISCSI_INITIATOR=%s", iqn(p.base, p.hostname)),
		fmt.Sprintf("ISCSI_TARGET_NAME=%s", p.target),
		fmt.Sprintf("ISCSI_TARGET_IP=%s", p.iscsiIP),
	)
	c.Append = strings.Join(args, " ")

	t, err := template.New("pxe").Parse(pxeTemplate)
	if err != nil {
		return errors.Wrap(err, "create pxe template")
	}
	return t.Execute(w, c)
}

func iqn(base, hostname string) string {
	return fmt.Sprintf(iqnFmt, year, base, hostname)
}

func pxeFilename(mac string) string {
	return fmt.Sprintf("01-%s", strings.Replace(mac, ":", "-", -1))
}

const (
	kvFmt  = "%s-%s"
	kernel = "vmlinuz"
	initrd = "initrd.img"
	year   = 2020
	iqnFmt = "iqn.%d.%s:%s"

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

func copyKernel(source, version, target string) error {
	// rename kernel images
	for _, name := range []string{initrd, kernel} {
		sourceFile := filepath.Join(source, name)
		fn := filepath.Join(source, fmt.Sprintf(kvFmt, name, version))
		if err := os.Rename(sourceFile, fn); err != nil {
			return errors.Wrap(err, "rename kernels to target")
		}
		sf, err := os.Open(fn)
		if err != nil {
			return err
		}

		fn = filepath.Join(target, fmt.Sprintf(kvFmt, name, version))
		f, err := os.Create(fn)
		if err != nil {
			sf.Close()
			return err
		}
		_, err = io.Copy(f, sf)
		sf.Close()
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func syncDir(ctx context.Context, source, target string) error {
	cmd := exec.CommandContext(ctx, "rsync", "--progress", "-a", source, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to rsync directories")
	}
	return nil
}

func getVersion(i string) string {
	parts := strings.Split(i, ":")
	return parts[len(parts)-1]
}
