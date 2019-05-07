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

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

const (
	dnsFormat     = "stellarproject.io/dns.%s"
	serviceFormat = "stellarproject.io/container.%s"
)

type service struct {
	Name string
	IP   string
	Port int64
}

func fetchServices(pool *redis.Pool, id string) ([]service, error) {
	conn := pool.Get()
	defer conn.Close()

	values, err := redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf(dnsFormat, id)))
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, errors.Errorf("service %q does not exist", id)
	}
	var out []service
	for _, name := range values {
		key := fmt.Sprintf(serviceFormat, id)
		ip, err := redis.String(conn.Do("HGET", key, "ip"))
		if err != nil {
			return nil, err
		}
		port, err := redis.Int64(conn.Do("HGET", key, fmt.Sprintf("service:%s", name)))
		if err != nil {
			return nil, err
		}
		out = append(out, service{
			Name: name,
			Port: port,
			IP:   ip,
		})
	}
	return out, nil
}
