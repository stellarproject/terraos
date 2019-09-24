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
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/stellarproject/terraos/version"
	"github.com/urfave/cli"
)

type Config map[string]interface{}

var configKeys = []string{
	"base",
	"pxe",
	"registry",
}

func (c Config) String(k string) string {
	v := c[k]
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

var configCommand = cli.Command{
	Name:  "config",
	Usage: "perform config operations",
	Subcommands: []cli.Command{
		configInitCommand,
		configGetCommand,
		configSetCommand,
	},
	Action: func(clix *cli.Context) error {
		fmt.Println("keys:")
		for _, k := range configKeys {
			fmt.Println(k)
		}
		return nil
	},
}

var configGetCommand = cli.Command{
	Name:  "get",
	Usage: "get a config value",
	Action: func(clix *cli.Context) error {
		value := clix.Args().First()
		if value == "" {
			return errors.New("value name not specified")
		}
		c, err := loadConfig()
		if err != nil {
			return err
		}
		fmt.Print(c.String(value))
		return nil
	},
}

var configSetCommand = cli.Command{
	Name:  "set",
	Usage: "set a config value",
	Action: func(clix *cli.Context) error {
		key := clix.Args().First()
		if key == "" {
			return errors.New("key not specified")
		}
		value := clix.Args().Get(1)
		c, err := loadConfig()
		if err != nil {
			return err
		}
		c[key] = value
		return saveConfig(c)
	},
}

var configInitCommand = cli.Command{
	Name:  "init",
	Usage: "init a new config",
	Action: func(clix *cli.Context) error {
		c := Config{}
		c["base"] = fmt.Sprintf("terraos:%s", version.Version)
		c["pxe"] = fmt.Sprintf("pxe:%s", version.Version)

		return saveConfig(c)
	},
}

func loadConfig() (Config, error) {
	var m map[string]interface{}
	if _, err := toml.DecodeFile("cluster.toml", &m); err != nil {
		return nil, err
	}
	return Config(m), nil
}

func saveConfig(c Config) error {
	f, err := os.Create("cluster.toml")
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}
