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

package remotes

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var plainRemotes = make(map[string]struct{})

const DefaultPath = "/cluster/plain-remotes.json"

func LoadRemotes(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "open plain remotes %s", path)
	}
	defer f.Close()
	var remotes []string
	if err := json.NewDecoder(f).Decode(&remotes); err != nil {
		return errors.Wrap(err, "decode plain remotes file")
	}
	SetRemotes(remotes)
	return nil
}

func SetRemotes(remotes []string) {
	for _, r := range remotes {
		plainRemotes[r] = struct{}{}
	}
}

func Plain(ref string) bool {
	u, err := url.Parse("registry://" + ref)
	if err != nil {
		logrus.WithError(err).Errorf("parse ref %s", ref)
		return false
	}
	plain := strings.Contains(u.Host, ":5000")
	if !plain {
		if _, ok := plainRemotes[u.Host]; ok {
			plain = true
		}
	}
	return plain
}

func WithPlainRemote(ref string) containerd.RemoteOpt {
	return func(_ *containerd.Client, ctx *containerd.RemoteContext) error {
		ctx.Resolver = docker.NewResolver(docker.ResolverOptions{
			PlainHTTP: Plain(ref),
			Client:    http.DefaultClient,
		})
		return nil
	}
}
