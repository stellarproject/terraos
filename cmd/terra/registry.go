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
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// RegistryAuth is the base64 encoded credentials for the registry credentials
type RegistryAuth struct {
	Auth string `json:"auth,omitempty"`
}

// DockerConfig is the docker config struct
type DockerConfig struct {
	Auths map[string]RegistryAuth `json:"auths"`
}

func getDockerCredentials(host string) (string, string, error) {
	logrus.WithField("host", host).Debug("checking for registry auth config")
	var (
		home             = os.Getenv("HOME")
		credentialConfig = filepath.Join(home, ".docker", "config.json")
	)
	f, err := os.Open(credentialConfig)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}
		return "", "", err
	}
	defer f.Close()

	var cfg DockerConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return "", "", err
	}

	for h, r := range cfg.Auths {
		if h == host {
			creds, err := base64.StdEncoding.DecodeString(r.Auth)
			if err != nil {
				return "", "", err
			}
			parts := strings.SplitN(string(creds), ":", 2)
			logrus.Debugf("using auth for registry %s: user=%s", host, parts[0])
			return parts[0], parts[1], nil
		}
	}

	return "", "", nil
}
