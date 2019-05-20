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

package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/stellarproject/terraos/pkg/store"
)

type FS struct {
	basePath string
}

func NewFS(basePath string) (*FS, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}
	return &FS{
		basePath: basePath,
	}, nil
}

func (f *FS) resolveKey(key string) string {
	return filepath.Join(f.basePath, key)
}

func (f *FS) mkPath(key string) error {
	return os.MkdirAll(filepath.Join(f.basePath, filepath.Dir(key)), 0755)
}

func (f *FS) Get(key string) (store.Item, error) {
	p := f.resolveKey(key)
	if _, err := os.Stat(p); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return nil, store.ErrNotFound
	}
	return os.Open(p)
}

func (f *FS) GetOrCreate(key string) (store.Item, error) {
	p := f.resolveKey(key)
	fmt.Println("------", p)
	if _, err := os.Stat(p); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		fmt.Println("create", p)
		f, err := os.Create(p)
		if err != nil {
			return nil, err
		}
		f.Close()
	}
	return f.Get(p)
}

func (f *FS) Set(key string, v []byte) error {
	if err := f.mkPath(key); err != nil {
		return err
	}
	p := f.resolveKey(key)
	x, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	if _, err := x.Write(v); err != nil {
		return err
	}
	return nil
}

func (f *FS) Delete(key string) error {
	p := f.resolveKey(key)
	if _, err := os.Stat(p); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		return store.ErrNotFound
	}
	return os.RemoveAll(p)
}

func (f *FS) List(key string) ([]store.Stat, error) {
	p := f.resolveKey(key)
	var fi []store.Stat
	info, err := ioutil.ReadDir(p)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return nil, store.ErrNotFound
	}
	for _, f := range info {
		fi = append(fi, f)
	}
	return fi, nil
}
