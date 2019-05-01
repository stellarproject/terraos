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
	"io"
	"strings"
	"text/template"
)

const (
	defaultRepo = "docker.io/stellarproject"
)

func joinImage(i, name, version string) string {
	return fmt.Sprintf("%s/%s:%s", i, name, version)
}

func cname(c Component) string {
	return c.Name
}

func cmdargs(args []string) string {
	return strings.Join(args, " ")
}

func imageName(c Component) string {
	return joinImage(defaultRepo, c.Name, c.Version)
}

func render(w io.Writer, tmp string, ctx *OSContext) error {
	t, err := template.New("dockerfile").Funcs(template.FuncMap{
		"cname":     cname,
		"imageName": imageName,
		"cmdargs":   cmdargs,
	}).Parse(tmp)
	if err != nil {
		return err
	}
	return t.Execute(w, ctx)
}
