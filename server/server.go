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

package server

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	v1 "github.com/stellarproject/terraos/api/terra/v1"
	v1types "github.com/stellarproject/terraos/api/types/v1"
)

func New(store *Store) (*Server, error) {
	return &Server{
		store: store,
	}, nil
}

type Server struct {
	store *Store
}

func (s *Server) Register(ctx context.Context, r *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	m := &v1types.Machine{
		UUID:           uuid.New().String(),
		Cpus:           r.Cpus,
		Memory:         r.Memory,
		NetworkDevices: r.NetworkDevices,
	}
	if err := s.store.Machines().Save(ctx, m); err != nil {
		return nil, err
	}
	return &v1.RegisterResponse{
		Machine: m,
	}, nil
}

func (s *Server) Machines(ctx context.Context, _ *types.Empty) (*v1.MachinesResponse, error) {
	machines, err := s.store.Machines().List(ctx)
	if err != nil {
		return nil, err
	}
	var resp v1.MachinesResponse
	for _, m := range machines {
		resp.Machines = append(resp.Machines, m)
	}
	return &resp, nil
}
