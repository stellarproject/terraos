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

package fstab

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Entry struct {
	Device  string
	Path    string
	Type    string
	Options []string
	Dump    bool
	Pass    int
}

func Write(w io.Writer, entries []*Entry) error {
	tw := tabwriter.NewWriter(w, 10, 1, 3, '	', 0)
	const tfmt = "%s\t%s\t%s\t%s\t%d\t%d\n"
	for _, e := range entries {
		dump := 0
		if e.Dump {
			dump = 1
		}
		fmt.Fprintf(tw, tfmt,
			e.Device,
			e.Path,
			e.Type,
			strings.Join(e.Options, ","),
			dump,
			e.Pass,
		)
	}
	return tw.Flush()
}
