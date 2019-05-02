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
	"strings"
	"text/template"
)

type PXE struct {
	IQN      string `toml:"iqn"`
	Target   string `toml:"target"`
	TargetIP string `toml:"target_ip"`
	Root     string `toml:"root"`
	IP       string `toml:"ip"`
	MAC      string `toml:"mac"`
}

type PXEContext struct {
	Append []string
}

const pxeTemplate = `DEFAULT terra

LABEL terra
  KERNEL /vmlinuz
  INITRD /initrd.img
  APPEND {{cmdargs .Append}}`

func setupPXE(id string, pxe *PXE) error {
	if pxe.IP == "" {
		pxe.IP = "dhcp"
	}
	args := []string{
		"boot=terra",
		fmt.Sprintf("ip=%s", pxe.IP),
	}
	if pxe.Root != "" {
		args = append(args, fmt.Sprintf("root=%s", pxe.Root))
	}
	if pxe.IQN != "" {
		args = append(args, fmt.Sprintf("ISCSI_INITIATOR=%s.%s", pxe.IQN, id))
	}
	if pxe.Target != "" {
		args = append(args,
			fmt.Sprintf("ISCSI_TARGET_NAME=%s:%s.%s", pxe.IQN, pxe.Target, id),
			fmt.Sprintf("ISCSI_TARGET_IP=%s", pxe.TargetIP),
		)
	}
	ctx := &PXEContext{
		Append: args,
	}

	t, err := template.New("pxe").Funcs(template.FuncMap{
		"cname":     cname,
		"imageName": imageName,
		"cmdargs":   cmdargs,
	}).Parse(pxeTemplate)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("01-%s", strings.Replace(pxe.MAC, ":", "-", -1))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, ctx)
}
