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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

type service struct {
	Name string
	IP   string
	Port int
}

func fetchServices(id string) ([]service, error) {
	path := filepath.Join("/cluster/service", id, "containers")
	containers, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Errorf("service %q does not exist", id)
		}
		return nil, errors.Wrapf(err, "read %s", path)
	}
	rport, err := ioutil.ReadFile(filepath.Join("/cluster/service", id, "port"))
	if err != nil {
		return nil, errors.Wrap(err, "read port")
	}
	port, err := strconv.Atoi(string(rport))
	if err != nil {
		return nil, errors.Wrap(err, "convert port to int")
	}
	var out []service
	for _, i := range containers {
		name := i.Name()
		ipp := filepath.Join(path, name, "ip")

		ip, err := ioutil.ReadFile(ipp)
		if err != nil {
			return nil, errors.Wrapf(err, "read ip from %s", ipp)
		}
		out = append(out, service{
			Name: name,
			Port: port,
			IP:   string(ip),
		})
	}
	return out, nil
}
