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

package pxe

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const DHCP = "dhcp"

const pxeTemplate = `DEFAULT {{.Default}}

{{range $e := .Entries}}
LABEL {{$e.Label}}
  KERNEL {{$e.Kernel}}
  INITRD {{$e.Initrd}}
  APPEND {{args $e}}
{{end}}`

type PXE struct {
	Default      string
	MAC          string
	InitiatorIQN string
	TargetIQN    string
	TargetIP     string
	IP           string

	Entries []Entry
}

type Entry struct {
	Root   string
	Boot   string
	Append []string

	Label  string
	Kernel string
	Initrd string
}

func (p *PXE) Filename() string {
	return fmt.Sprintf("01-%s", strings.Replace(p.MAC, ":", "-", -1))
}

func (p *PXE) Write(w io.Writer) error {
	t, err := template.New("pxe").Funcs(template.FuncMap{
		"args": p.args,
	}).Parse(pxeTemplate)
	if err != nil {
		return errors.Wrap(err, "create pxe template")
	}
	return t.Execute(w, p)
}

func (p *PXE) args(e Entry) string {
	args := []string{
		fmt.Sprintf("ip=%s", p.IP),
		fmt.Sprintf("boot=%s", e.Boot),
	}
	if e.Root != "" {
		args = append(args, fmt.Sprintf("root=%s", e.Root))
	}
	if p.InitiatorIQN != "" {
		args = append(args, fmt.Sprintf("ISCSI_INITIATOR=%s", p.InitiatorIQN))
	}
	if p.TargetIQN != "" {
		args = append(args,
			fmt.Sprintf("ISCSI_TARGET_NAME=%s", p.TargetIQN),
			fmt.Sprintf("ISCSI_TARGET_IP=%s", p.TargetIP),
		)
	}
	args = append(args, e.Append...)
	return strings.Join(args, " ")
}
