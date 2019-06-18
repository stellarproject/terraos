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

package v1

import (
	"fmt"

	"github.com/stellarproject/terraos/pkg/fstab"
	"github.com/stellarproject/terraos/pkg/mkfs"
)

const (
	year       = 2019
	nodeIqnFmt = "iqn.%d.%s:%s"
	initIQN    = "iqn.%d.%s:%s.%s"
)

func (n *Node) InitiatorIQN() string {
	return fmt.Sprintf(initIQN, year, n.Domain, "node", n.Hostname)
}

func (n *Node) IQN() string {
	return fmt.Sprintf(nodeIqnFmt, year, n.Domain, n.Hostname)
}

func (v *Volume) Entries() []*fstab.Entry {
	var entries []*fstab.Entry
	if len(v.Subvolumes) > 0 {
		for _, s := range v.Subvolumes {
			options := []string{
				fmt.Sprintf("subvol=/%s", s.Name),
			}
			if !s.Cow {
				options = append(options, "nodatacow")
			}
			entries = append(entries, &fstab.Entry{
				Type:    mkfs.Btrfs,
				Device:  fmt.Sprintf("LABEL=%s", v.Label),
				Path:    s.Path,
				Options: options,
			})
		}
		return entries
	}
	return []*fstab.Entry{
		&fstab.Entry{
			Type:   v.FsType,
			Pass:   2,
			Device: fmt.Sprintf("LABEL=%s", v.Label),
			Path:   v.Path,
		},
	}
}
