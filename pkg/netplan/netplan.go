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

package netplan

import (
	"io"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const (
	DefaultVersion     = 2
	DefaultRenderer    = "networkd"
	DefaultFilename    = "01-netcfg.yaml"
	InterfacesFilename = "interfaces"
)

const interfacesTemplate = `auto lo
iface lo inet loopback

{{range $i := .Interfaces}}
auto {{$i.Name}}
iface {{$i.Name}} inet {{inet $i}}
	{{if ne $i.Gateway ""}}gateway {{$i.Gateway}}{{end}}
{{end}}
`

const netplanTemplate = `network:
  version: 2
  renderer: networkd
  ethernets:{{range $i := .Interfaces}}
    {{ $i.Name }}:
      {{if $i.Addresses}}addresses: [{{addresses $i.Addresses}}]{{else}}dhcp4: yes{{end}}{{if ne $i.Gateway ""}}
      gateway4: {{$i.Gateway}}{{end}}{{end}}`

type Netplan struct {
	Interfaces []Interface `toml:"interfaces"`
}

func (n *Netplan) WriteInterfaces(w io.Writer) error {
	t, err := template.New("interfaces").Funcs(template.FuncMap{
		"addresses":   addresses,
		"nameservers": nameservers,
		"inet":        netType,
	}).Parse(interfacesTemplate)
	if err != nil {
		return errors.Wrap(err, "interfaces template")
	}
	return t.Execute(w, n)
}

func (n *Netplan) Write(w io.Writer) error {
	t, err := template.New("netplan").Funcs(template.FuncMap{
		"addresses":   addresses,
		"nameservers": nameservers,
	}).Parse(netplanTemplate)
	if err != nil {
		return errors.Wrap(err, "netplan template")
	}
	return t.Execute(w, n)
}

type Interface struct {
	Name      string   `toml:"name"`
	Addresses []string `toml:"addresses"`
	Gateway   string   `toml:"gateway"`
}

func addresses(v []string) string {
	return strings.Join(v, ",")
}

func nameservers(v []string) string {
	return strings.Join(v, ",")
}

func netType(i Interface) string {
	if len(i.Addresses) > 0 {
		return "static"
	}
	return "dhcp"
}
