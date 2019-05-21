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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/stellarproject/terraos/pkg/store"
)

var (
	ErrNoFile = errors.New("no such file or directory")
)

type Dir interface {
	os.FileInfo
}

type File interface {
	io.ReadWriteCloser
	os.FileInfo
}

type file struct {
	name    string
	path    string
	uid     string
	gid     string
	isDir   bool
	data    []byte
	backend store.Store
}

func newFile(name, path, uid, gid string, isDir bool, data []byte, backend store.Store) (*file, error) {
	if name == "" || path == "" {
		return nil, fmt.Errorf("name and path must be specified")
	}
	if uid == "" {
		uid = "0"
	}
	if gid == "" {
		gid = "0"
	}
	if backend == nil {
		return nil, fmt.Errorf("backend must not be nil")
	}
	return &file{
		name:    name,
		path:    path,
		data:    data,
		uid:     uid,
		gid:     gid,
		isDir:   isDir,
		backend: backend,
	}, nil
}

func (f *file) ReadAt(p []byte, off int64) (int, error) {
	// read data direct
	size := len(f.data)
	if size > 0 {
		if off > int64(size) {
			return 0, io.EOF
		}
		n := copy(p, f.data[off:])
		return n, nil
	}
	// lookup from store
	x, err := f.backend.Get(f.path)
	if err != nil {
		return -1, err
	}
	data, err := ioutil.ReadAll(x)
	if err != nil {
		return -1, err
	}
	n := copy(p, data[off:])
	return n, nil
}
func (f *file) Name() string {
	return f.name
}
func (f *file) Size() int64 {
	if f.isDir {
		return 0
	}
	if s := len(f.data); s > 0 {
		return int64(s)
	}
	// get from store
	x, err := f.backend.Get(f.path)
	if err != nil {
		return 0
	}
	data, err := ioutil.ReadAll(x)
	if err != nil {
		return -1
	}
	return int64(len(data))
}
func (f *file) Sys() interface{} {
	return f
}
func (f *file) IsDir() bool {
	return f.isDir
}
func (f *file) Uid() string {
	return f.uid
}
func (f *file) Gid() string {
	return f.gid
}
func (f *file) Muid() string {
	return f.uid
}
func (f *file) ModTime() time.Time {
	return time.Now()
}
func (f *file) Mode() os.FileMode {
	if f.isDir {
		return os.ModeDir | 0755
	}
	return 0644
}
func (f *file) WriteAt(p []byte, off int64) (int, error) {
	// TODO offset
	if err := f.backend.Set(f.path, p); err != nil {
		return -1, err
	}
	return len(p), nil
}
func (f *file) Close() error {
	return nil
}

type dir struct {
	name    string
	mode    os.FileMode
	entries []os.FileInfo
	c       chan os.FileInfo
	done    chan struct{}
}

func (d *dir) Mode() os.FileMode  { return os.ModeDir | d.mode }
func (d *dir) IsDir() bool        { return true }
func (d *dir) Name() string       { return d.name }
func (d *dir) ModTime() time.Time { return time.Now() }
func (d *dir) Size() int64        { return 4096 }
func (d *dir) Sys() interface{}   { return d }

func (d *dir) Readdir(n int) ([]os.FileInfo, error) {
	var err error
	fi := make([]os.FileInfo, 0, 10)
	for i := 0; i < n; i++ {
		s, ok := <-d.c
		if !ok {
			err = io.EOF
			break
		}
		fi = append(fi, s)
	}
	return fi, err
}

func (d *dir) Close() error {
	close(d.done)
	return nil
}

func mkdir(entries []os.FileInfo, mode os.FileMode) *dir {
	c := make(chan os.FileInfo, 10)
	done := make(chan struct{})
	d := &dir{
		entries: entries,
		mode:    mode,
	}
	go func() {
		for _, v := range d.entries {
			select {
			case c <- v:
			case <-done:
				break
			}
		}
		close(c)
	}()
	d.c = c
	d.done = done
	return d
}
