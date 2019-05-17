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

package configstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gomodule/redigo/redis"
)

const configFmt = "stellarproject.io/config:%s"

type ConfigFile struct {
	ID      string `json:"id" toml:"id"`
	Content string `json:"content" toml:"content"`
	Signal  string `json:"signal" toml:"signal"`
}

func (c *ConfigFile) Write(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(c.Content)
	return err
}

func New(pool *redis.Pool) *ConfigStore {
	return &ConfigStore{
		pool: pool,
	}
}

type ConfigStore struct {
	pool *redis.Pool
}

func (c *ConfigStore) Create(ctx context.Context, config *ConfigFile) error {
	conn := c.pool.Get()
	defer conn.Close()

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if _, err := conn.Do("SETNX", fmt.Sprintf(configFmt, config.ID), data); err != nil {
		return err
	}
	return nil
}

func (c *ConfigStore) Get(ctx context.Context, ids []string) ([]*ConfigFile, error) {
	conn := c.pool.Get()
	defer conn.Close()

	var out []*ConfigFile
	for _, id := range ids {
		data, err := redis.Bytes(conn.Do("GET", id))
		if err != nil {
			return nil, err
		}
		var v ConfigFile
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, nil
}
