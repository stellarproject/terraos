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
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// this package includes global redis keys and functions
const (
	ConfigsKey    = "terra.config.*"
	ConfigFmtKey  = "terra.config.%s"
	VolumesKey    = "terra.volumes.*"
	VolumeFmtKey  = "terra.volumes.%s"
	MachineFmtKey = "terra.machines.%s"
	MachinesKey   = "terra.machines.*"
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

func (s *Store) Machines() *MachineStore {
	return &MachineStore{
		s: s,
	}
}

type MachineStore struct {
	s *Store
}

func (s *MachineStore) List(ctx context.Context) ([]*Machine, error) {
	keys, err := redis.Strings(s.s.do(ctx, "KEYS", MachinesKey))
	if err != nil {
		return nil, err
	}
	var out []*Machine
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
		out = append(out, c)
	}
	return out, nil
}

func (s *MachineStore) Register(ctx context.Context, c *Machine) error {
	c.UUID = uuid.New().String()
	return s.Save(ctx, c)
}

func (s *MachineStore) Save(ctx context.Context, c *Machine) error {
	key := fmt.Sprintf(MachineFmtKey, c.UUID)
	return s.s.setProto(ctx, key, c)
}

func (s *MachineStore) Get(ctx context.Context, id string) (*Machine, error) {
	key := fmt.Sprintf(MachineFmtKey, id)
	data, err := redis.Bytes(s.s.do(ctx, "GET", key))
	if err != nil {
		return nil, errors.Wrap(err, "get machine")
	}
	var m Machine
	if err := proto.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

type ConfigStore struct {
	s *Store
}

func (s *ConfigStore) List(ctx context.Context) ([]*Config, error) {
	keys, err := redis.Strings(s.s.do(ctx, "KEYS", ConfigsKey))
	if err != nil {
		return nil, err
	}
	var out []*Config
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
		out = append(out, c)
	}
	return out, nil
}

func (s *ConfigStore) Save(ctx context.Context, c *Config) error {
	key := fmt.Sprintf(ConfigFmtKey, c.ID)
	return s.s.setProto(ctx, key, c)
}

func (s *ConfigStore) Get(ctx context.Context, id string) (*Config, error) {
	key := fmt.Sprintf(ConfigFmtKey, id)
	data, err := redis.Bytes(s.s.do(ctx, "GET", key))
	if err != nil {
		return nil, err
	}
	var c Config
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

type VolumeStore struct {
	s *Store
}

func (s *VolumeStore) Save(ctx context.Context, v *Volume) error {
	key := fmt.Sprintf(VolumeFmtKey, v.ID)
	return s.s.setProto(ctx, key, v)
}

func (s *VolumeStore) List(ctx context.Context) ([]*Volume, error) {
	keys, err := redis.Strings(s.s.do(ctx, "KEYS", VolumesKey))
	if err != nil {
		return nil, err
	}
	var out []*Volume
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
		out = append(out, c)
	}
	return out, nil
}

func (s *VolumeStore) Get(ctx context.Context, id string) (*Volume, error) {
	key := fmt.Sprintf(VolumeFmtKey, id)
	data, err := redis.Bytes(s.s.do(ctx, "GET", key))
	if err != nil {
		return nil, err
	}
	var v Volume
	if err := proto.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Store) do(ctx context.Context, action string, args ...interface{}) (interface{}, error) {
	conn, err := s.pool.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.Do(action, args...)
}

func (s *Store) setProto(ctx context.Context, key string, v proto.Message) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.do(ctx, "SET", key, data)
	return err
}
