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
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

// this package includes global redis keys and functions
const (
	ClusterKey   = "terra.cluster"
	ConfigsKey   = "terra.config.*"
	ConfigFmtKey = "terra.config.%s"
	VolumesKey   = "terra.volumes.*"
	VolumeFmtKey = "terra.volumes.%s"
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

func (s *Store) Configs() *ConfigStore {
	return &ConfigStore{
		s: s,
	}
}

func (s *Store) Volumes() *VolumeStore {
	return &VolumeStore{
		s: s,
	}
}

type ConfigStore struct {
	s *Store
}

func (s *ConfigStore) List(ctx context.Context) (map[string]*Config, error) {
	keys, err := redis.Strings(s.s.do(ctx, "KEYS", ConfigsKey))
	if err != nil {
		return nil, err
	}
	out := make(map[string]*Config, len(keys))
	for _, key := range keys {
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 3 {
			return nil, errors.New("invalid key type")
		}
		id := parts[2]
		c, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, nil
}

func (s *ConfigStore) Save(ctx context.Context, id string, c *Config) error {
	key := fmt.Sprintf(ConfigFmtKey, id)
	if _, err := s.s.do(ctx, "HMSET", key, "path", c.Path, "contents", c.Contents); err != nil {
		return errors.Wrap(err, "set config values")
	}
	return nil
}

func (s *ConfigStore) Get(ctx context.Context, id string) (*Config, error) {
	key := fmt.Sprintf(ConfigFmtKey, id)
	values, err := redis.Values(s.s.do(ctx, "HGETALL", key))
	if err != nil {
		return nil, errors.Wrap(err, "get all config fields")
	}
	var c Config
	for i := 0; i < len(values); i += 2 {
		switch string(values[i].([]byte)) {
		case "path":
			c.Path = string(values[i+1].([]byte))
		case "contents":
			c.Contents = values[i+1].([]byte)
		}
	}
	return &c, nil
}

type VolumeStore struct {
	s *Store
}

func (s *VolumeStore) Save(ctx context.Context, id string, v *Volume) error {
	key := fmt.Sprintf(VolumeFmtKey, id)
	args := []interface{}{
		key,
	}
	for i, l := range v.Luns {
		args = append(args,
			fmt.Sprintf("lun[%d].path", i), l.Path,
			fmt.Sprintf("lun[%d].label", i), l.Label,
		)
	}
	if _, err := s.s.do(ctx, "HMSET", args...); err != nil {
		return err
	}
	return nil
}

func (s *VolumeStore) List(ctx context.Context) (map[string]*Volume, error) {
	keys, err := redis.Strings(s.s.do(ctx, "KEYS", VolumesKey))
	if err != nil {
		return nil, err
	}
	out := make(map[string]*Volume, len(keys))
	for _, key := range keys {
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 3 {
			return nil, errors.New("invalid key type")
		}
		id := parts[2]
		c, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, nil
}

func (s *VolumeStore) Get(ctx context.Context, id string) (*Volume, error) {
	key := fmt.Sprintf(VolumeFmtKey, id)
	values, err := redis.StringMap(s.s.do(ctx, "HGETALL", key))
	if err != nil {
		return nil, err
	}
	var v Volume
	for i := 0; i <= 8; i++ {
		path, ok := values[fmt.Sprintf("lun[%d].path", i)]
		if !ok {
			break
		}
		label := values[fmt.Sprintf("lun[%d].label", i)]
		v.Luns = append(v.Luns, &Lun{
			Path:  path,
			Label: label,
		})
	}
	return &v, nil
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
	// reset fields
	c.Generation = 0
	c.Sha256 = ""
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
