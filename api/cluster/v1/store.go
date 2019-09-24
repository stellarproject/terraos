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
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

// this package includes global redis keys and functions
const (
	ClusterKey = "terra.cluster"
)

func New(address, auth string) *Store {
	pool := redis.NewPool(func() (redis.Conn, error) {
		conn, err := redis.Dial("tcp", address)
		if err != nil {
			return nil, errors.Wrap(err, "dial redis")
		}
		if auth != "" {
			if _, err := conn.Do("AUTH", auth); err != nil {
				conn.Close()
				return nil, errors.Wrap(err, "authenticate to redis")
			}
		}
		return conn, nil
	}, 10)
	return &Store{
		pool: pool,
	}
}

type Store struct {
	pool *redis.Pool
}

func (s *Store) Close() error {
	return s.pool.Close()
}

func (s *Store) Get(ctx context.Context) (*Cluster, error) {
	data, err := redis.Bytes(s.do(ctx, "GET", ClusterKey))
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch cluster")
	}
	var c Cluster
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal cluster")
	}
	return &c, nil
}

func (s *Store) Commit(ctx context.Context, c *Cluster) error {
	data, err := proto.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "marshal cluster")
	}
	/*
		sha := sha256.New()
		if _, err := sha.Write(data); err != nil {
			return errors.Wrap(err, "hash cluster")
		}
	*/

	if _, err := s.do(ctx, "SET", ClusterKey, data); err != nil {
		return errors.Wrap(err, "commit cluster")
	}
	return nil
}

func (s *Store) do(ctx context.Context, action string, args ...interface{}) (interface{}, error) {
	conn, err := s.pool.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.Do(action, args...)
}
