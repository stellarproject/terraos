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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"aqwari.net/net/styx"
	"github.com/stellarproject/terraos/pkg/store"
)

func clusterHandler(backend store.Store) handler {
	return func(p string, r styx.Request) (interface{}, interface{}, error) {
		fmt.Printf("clusterHandler: path=%q\n", p)
		switch p {
		case "":
			return nil, mkdir([]os.FileInfo{
				&dir{name: "services", mode: 0775},
				&dir{name: "nodes", mode: 0755},
			}), nil
		case "/services":
			// TODO: pull from store
			info, err := backend.List(r.Path())
			if err != nil {
				if !store.IsNotFound(err) {
					return nil, nil, err
				}
			}
			fi, err := fromStat(info, backend)
			if err != nil {
				return nil, nil, err
			}
			return nil, mkdir(fi), nil
		case "/nodes":
			// TODO: pull from store
			info, err := backend.List(r.Path())
			if err != nil {
				if !store.IsNotFound(err) {
					return nil, nil, err
				}
			}
			fi, err := fromStat(info, backend)
			if err != nil {
				return nil, nil, err
			}
			return nil, mkdir(fi), nil
		default:
			switch v := r.(type) {
			case styx.Tcreate:
				f, err := backend.GetOrCreate(r.Path())
				if err != nil {
					return nil, nil, err
				}
				return nil, f, nil
			case styx.Twalk:
				info, err := backend.List(path.Dir(r.Path()))
				if err != nil {
					return nil, nil, err
				}
				fmt.Printf("clusterHandler: %+v\n", info)
				fi, err := fromStat(info, backend)
				if err != nil {
					return nil, nil, err
				}
				return nil, mkdir(fi), nil
			default:
				fmt.Printf("clusterHandler: default %T\n", v)
				f, err := backend.Get(r.Path())
				if err != nil {
					return nil, nil, err
				}
				fmt.Printf("clusterHandler default backend.Get %T\n", f)
				fi, err := toFile(f)
				if err != nil {
					return nil, nil, err
				}
				return nil, fi, nil
			}
			return nil, nil, ErrNoFile
		}
		return nil, nil, ErrNoFile
	}
}

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

func toFile(i store.Item) (*file, error) {
	f := i.(*os.File)
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &file{
		name: filepath.Base(f.Name()),
		data: data,
	}, nil
}
