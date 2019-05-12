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
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const netplanTemplate = `network:
  version: 2
  renderer: networkd
  ethernets:
    {{ .Interface }}:
      {{if .Addresses}}addresses: [{{addresses .Addresses}}]{{else}}dhcp4: yes{{end}}
      {{if ne .Gateway ""}}gateway4: {{.Gateway}}{{end}}`

type Netplan struct {
	Interface string   `toml:"interface"`
	Addresses []string `toml:"addresses"`
	Gateway   string   `toml:"gateway"`
}

func setupNetplan(path string, n Netplan, paths *[]string) error {
	if n.Interface == "" {
		n.Interface = "eth0"
	}
	t, err := template.New("netplan").Funcs(template.FuncMap{
		"addresses":   addresses,
		"nameservers": nameservers,
	}).Parse(netplanTemplate)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(path, "01-netcfg.yaml"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	*paths = append(*paths, f.Name())
	defer f.Close()
	return t.Execute(f, n)
}

func addresses(v []string) string {
	return strings.Join(v, ",")
}

func nameservers(v []string) string {
	return strings.Join(v, ",")
}
