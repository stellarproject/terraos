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

package resolvconf

import (
	"fmt"
	"io"
)

// Default resolv.conf path
const DefaultPath = "/etc/resolv.conf"

// DefaultNameservers are the google DNS servers
var DefaultNameservers = []string{
	"8.8.8.8",
	"8.8.4.4",
}

// Conf for the resolver
type Conf struct {
	Nameservers []string `toml:"nameservers"`
	Search      string   `toml:"search"`
}

// Write the conf to the provided path
func (r *Conf) Write(w io.Writer) error {
	for _, ns := range r.Nameservers {
		if _, err := fmt.Fprintf(w, "nameserver %s\n", ns); err != nil {
			return err
		}
	}
	if r.Search != "" {
		if _, err := fmt.Fprintf(w, "search %s", r.Search); err != nil {
			return err
		}
	}
	return nil
}
