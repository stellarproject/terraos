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

package cmd

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

const (
	ConfigPath     = "/cluster/terra.toml"
	DefaultRuntime = "io.containerd.runc.v2"
)

func Default() Terra {
	return Terra{
		Address: "0.0.0.0",
		Port:    9000,
		Redis:   "127.0.0.1:6379",
	}
}

type Terra struct {
	SentryDSN string `toml:"dns"`
	Redis     string `toml:"redis"`
	Debug     bool   `toml:"debug"`
	Address   string `toml:"address"`
	Port      int    `toml:"port"`
}

func (t *Terra) Addr() string {
	return fmt.Sprintf("%s:%d", t.Address, t.Port)
}

func LoadTerra() (*Terra, error) {
	t := Default()
	if _, err := toml.DecodeFile(ConfigPath, &t); err != nil {
		return &t, err
	}
	return &t, nil
}
