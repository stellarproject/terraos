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

package iscsi

import (
	"context"
	"errors"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/stellarproject/terraos/api/v1/types"
)

var (
	ErrTransactionDone = errors.New("transaction already done")
	ErrLunExists       = errors.New("lun exists")
	ErrTargetNotExist  = errors.New("target does not exist")
	ErrLUNNotExist     = errors.New("lun does not exist")
)

const TargetKey = "io.stellarproject.iscsi.targets"

type store struct {
	mu   sync.Mutex
	pool *redis.Pool
}

func (s *store) Close() error {
	return s.pool.Close()
}

func (s *store) Begin(ctx context.Context) (_ *Transaction, err error) {
	s.mu.Lock()
	defer func() {
		if err != nil {
			s.mu.Unlock()
		}
	}()
	conn, err := s.pool.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var state types.ISCSIState
	data, err := redis.Bytes(conn.Do("GET", TargetKey))
	if err != nil {
		if err == redis.ErrNil {
			return &Transaction{
				s:     s,
				State: &state,
			}, nil
		}
		return nil, err
	}
	if err := proto.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &Transaction{
		s:     s,
		State: &state,
	}, nil
}

type Transaction struct {
	State *types.ISCSIState

	mu   sync.Mutex
	s    *store
	done bool
}

func (t *Transaction) Commit(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		return ErrTransactionDone
	}
	data, err := proto.Marshal(t.State)
	if err != nil {
		return err
	}
	conn, err := t.s.pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.Do("SET", TargetKey, data); err != nil {
		return err
	}
	t.done = true
	t.s.mu.Unlock()
	return nil
}

func (t *Transaction) Rollback() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.done {
		return
	}
	t.done = true
	t.s.mu.Unlock()
}
