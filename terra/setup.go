package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
)

const hostsFile = `127.0.0.1       localhost %s
::1             localhost ip6-localhost ip6-loopback
ff02::1         ip6-allnodes
ff02::2         ip6-allrouters`

var setupCommand = cli.Command{
	Name:  "setup",
	Usage: "setup the vhost configuration",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "hostname",
			Usage: "set the hostname",
			Value: "terra",
		},
		cli.StringSliceFlag{
			Name:  "nameserver,n",
			Usage: "set the nameservers",
			Value: &cli.StringSlice{},
		},
		cli.StringFlag{
			Name:  "github",
			Usage: "github url to download keys for",
		},
	},
	Action: func(clix *cli.Context) error {
		config := filepath.Join(clix.GlobalString("root"), "config")
		var files []*file
		if hostname := clix.String("hostname"); hostname != "" {
			files = append(files, &file{
				path:     "etc/hostname",
				contents: hostname,
			},
				&file{
					path:     "etc/hosts",
					contents: fmt.Sprintf(hostsFile, hostname),
				},
			)
		}
		if ns := clix.StringSlice("nameserver"); len(ns) > 0 {
			var lines []string
			for _, n := range ns {
				lines = append(lines, fmt.Sprintf("nameserver %s", n))
			}
			files = append(files, &file{
				path:     "etc/resolv.conf",
				contents: strings.Join(lines, "\n"),
			})
		}
		if ssh := clix.String("github"); ssh != "" {
			r, err := http.Get(ssh)
			if err != nil {
				return err
			}
			defer r.Body.Close()
			files = append(files, &file{
				path: "home/terra/.ssh/authorized_keys",
				r:    r.Body,
			})
		}
		for _, f := range files {
			if err := f.write(config); err != nil {
				return err
			}
		}
		return nil
	},
}

type file struct {
	path     string
	contents string
	r        io.Reader
}

func (ff *file) write(base string) error {
	dir := filepath.Dir(filepath.Join(base, ff.path))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(base, ff.path))
	if err != nil {
		return err
	}
	defer f.Close()
	if ff.r != nil {
		_, err = io.Copy(f, ff.r)
	} else {
		_, err = f.WriteString(ff.contents)
	}
	return err
}
