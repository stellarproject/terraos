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

package galaxy

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/stellarproject/terraos/pkg/store"
)

func fromStat(items []store.Stat, backend store.Store) ([]os.FileInfo, error) {
	var fi []os.FileInfo
	for _, f := range items {
		if f.IsDir() {
			fi = append(fi, &dir{name: f.Name(), mode: 0775})
		} else {
			fi = append(fi, f)
		}
	}
	return fi, nil
}

func toFile(i store.Item, p string, b store.Store) (*file, error) {
	f := i.(*os.File)
	// TODO: move to file Read
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &file{
		name:    filepath.Base(f.Name()),
		path:    p,
		backend: b,
		data:    data,
	}, nil
}
