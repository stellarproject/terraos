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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containerd/containerd/content"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/stellarproject/terraos/api/v1/types"
	"github.com/stellarproject/terraos/cmd"
	"github.com/stellarproject/terraos/pkg/btrfs"
	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/image"
	"github.com/stellarproject/terraos/pkg/resolvconf"
	"github.com/stellarproject/terraos/pkg/syslinux"
	"github.com/stellarproject/terraos/stage0"
	"github.com/stellarproject/terraos/stage1"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
	"golang.org/x/sys/unix"
)

func main() {
	app := cli.NewApp()
	app.Name = "terra-install"
	app.Version = version.Version
	app.Usage = "[repo] [destination]"
	app.Description = `
                                                     ___
                                                  ,o88888
                                               ,o8888888'
                         ,:o:o:oooo.        ,8O88Pd8888"
                     ,.::.::o:ooooOoOoO. ,oO8O8Pd888'"
                   ,.:.::o:ooOoOoOO8O8OOo.8OOPd8O8O"
                  , ..:.::o:ooOoOOOO8OOOOo.FdO8O8"
                 , ..:.::o:ooOoOO8O888O8O,COCOO"
                , . ..:.::o:ooOoOOOO8OOOOCOCO"
                 . ..:.::o:ooOoOoOO8O8OCCCC"o
                    . ..:.::o:ooooOoCoCCC"o:o
                    . ..:.::o:o:,cooooCo"oo:o:
                 ` + "`" + `   . . ..:.:cocoooo"'o:o:::'
                 .` + "`" + `   . ..::ccccoc"'o:o:o:::'
                :.:.    ,c:cccc"':.:.:.:.:.'
              ..:.:"'` + "`" + `::::c:"'..:.:.:.:.:.'
            ...:.'.:.::::"'    . . . . .'
           .. . ....:."' ` + "`" + `   .  . . ''
         . . . ...."'
         .. . ."'
        .
Install terra onto a physical disk`
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.BoolFlag{
			Name:  "http",
			Usage: "fetch image over http",
		},
		cli.StringFlag{
			Name:  "gateway",
			Usage: "gateway ip",
		},
	}
	app.Before = func(clix *cli.Context) error {
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Action = func(clix *cli.Context) error {
		gateway := clix.GlobalString("gateway")
		if gateway == "" {
			return errors.New("--gateway not specified")
		}

		node, err := cmd.LoadNode(clix.Args().First())
		if err != nil {
			return errors.Wrap(err, "load node")
		}
		var (
			diskmount = "/tmp/mnt"
			dest      = "/tmp/dest"
			ctx       = cmd.CancelContext()
		)
		for _, p := range []string{diskmount, dest} {
			if err := os.MkdirAll(p, 0755); err != nil {
				return errors.Wrapf(err, "mkdir %s", p)
			}
		}

		var (
			entries []*fstab.Entry
			store   content.Store
		)
		for _, g := range node.DiskGroups {
			if g.Stage != v1.Stage1 {
				continue
			}
			path := filepath.Join(diskmount, g.Label)
			if err := os.MkdirAll(path, 0755); err != nil {
				return errors.Wrapf(err, "mkdir %s", path)
			}
			if err := stage0.Format(g); err != nil {
				return errors.Wrap(err, "format group")
			}

			// mount the entire diskmount group before subsystems
			if err := unix.Mount(g.Disks[0].Device, path, stage0.DefaultFilesystem, 0, ""); err != nil {
				return errors.Wrapf(err, "mount %s to %s", g.Disks[0].Device, path)
			}
			if store == nil {
				storePath := filepath.Join(path, "terra-install-content")
				if store, err = image.NewContentStore(storePath); err != nil {
					return errors.Wrap(err, "new content store")
				}
			}
			group, err := stage1.NewGroup(g, dest)
			if err != nil {
				return errors.Wrap(err, "new stage1 disk group")
			}
			defer group.Close()

			if err := group.Init(path, nil); err != nil {
				return err
			}
			entries = append(entries, group.Entries()...)
		}
		if store == nil {
			return errors.New("store not created on any group")
		}
		// install
		desc, err := image.Fetch(ctx, clix.GlobalBool("http"), store, node.Image)
		if err != nil {
			return errors.Wrap(err, "fetch image")
		}
		if err := image.Unpack(ctx, store, desc, dest); err != nil {
			return errors.Wrap(err, "unpack image")
		}

		for _, g := range node.DiskGroups {
			if g.Mbr {
				/*
					add the boot dir?
						entries = append(entries, &fstab.Entry{
							Type:   mkfs.Btrfs,
							Device: fmt.Sprintf("LABEL=%s", d.Label),
							Path:   "/boot",
							Pass:   2,
							Options: []string{
								"bind",
							},
						})
				*/

				path := filepath.Join(diskmount, g.Label)
				if err := syslinux.Copy(path); err != nil {
					return errors.Wrap(err, "copy syslinux from live cd")
				}
				if err := syslinux.InstallMBR(removePartition(g.Disks[0].Device), "/boot/syslinux/mbr.bin"); err != nil {
					return errors.Wrap(err, "install mbr")
				}
				if err := syslinux.ExtlinuxInstall(filepath.Join(path, "boot", "syslinux")); err != nil {
					return errors.Wrap(err, "install extlinux")
				}
			}
		}
		if err := writeFstab(entries, dest); err != nil {
			return errors.Wrap(err, "write fstab")
		}
		if err := writeResolvconf(dest, gateway); err != nil {
			return errors.Wrap(err, "write resolv.conf")
		}
		// snapshot the os install
		for _, g := range node.DiskGroups {
			if g.Label == stage1.OSLabel {
				var (
					path   = filepath.Join(diskmount, g.Label)
					source = filepath.Join(path, stage1.OSVolume)
					dest   = filepath.Join(path, stage1.SnapshotVolume)
				)
				if err := os.MkdirAll(dest, 0711); err != nil {
					return err
				}
				if err := btrfs.Snapshot(source, dest); err != nil {
					return err
				}
				break
			}
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func removePartition(device string) string {
	partition := string(device[len(device)-1])
	if _, err := strconv.Atoi(partition); err != nil {
		return device
	}
	if strings.Contains(device, "nvme") {
		partition = "p" + partition
	}
	return strings.TrimSuffix(device, partition)
}

func writeFstab(entries []*fstab.Entry, root string) error {
	f, err := os.Create(filepath.Join(root, fstab.Path))
	if err != nil {
		return errors.Wrap(err, "create fstab file")
	}
	defer f.Close()
	if err := fstab.Write(f, entries); err != nil {
		return errors.Wrap(err, "write fstab")
	}
	return nil
}

func writeResolvconf(root, gateway string) error {
	path := filepath.Join(root, resolvconf.DefaultPath)
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "create resolv.conf file %s", path)
	}
	defer f.Close()

	conf := &resolvconf.Conf{
		Nameservers: []string{
			gateway,
		},
	}
	return conf.Write(f)
}
