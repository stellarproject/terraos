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

package galaxy

import (
	"sort"
	"strings"

	"aqwari.net/net/styx"
	"github.com/stellarproject/terraos/pkg/store"
)

type matcher interface {
	Path() string
	Matches(path string) bool
	Handler() handler
}

type handler func(path string, r styx.Request) (interface{}, interface{}, error)

type router struct {
	handlers []matcher
	backend  store.Store
}

// pathMatcher matches on exact path
type pathMatcher struct {
	path    string
	handler handler
}

func (p *pathMatcher) Path() string {
	return p.path
}

func (p *pathMatcher) Matches(path string) bool {
	return p.path == path
}
func (p *pathMatcher) Handler() handler {
	return p.handler
}

// prefixMatcher matches on a path prefix
type prefixMatcher struct {
	prefix  string
	handler handler
}

func (p *prefixMatcher) Path() string {
	return p.prefix
}

func (p *prefixMatcher) Matches(path string) bool {
	v := strings.Index(path, p.prefix)
	return v >= 0
}
func (p *prefixMatcher) Handler() handler {
	return p.handler
}

func newRouter() *router {
	return &router{}
}

// Path adds a new route handler at the specific path
func (r *router) Path(p string, h handler) {
	r.handlers = append(r.handlers, &pathMatcher{
		path:    p,
		handler: h,
	})
}

// Prefix adds a new route handler using a prefix
func (r *router) Prefix(p string, h handler) {
	r.handlers = append(r.handlers, &prefixMatcher{
		prefix:  p,
		handler: h,
	})
}

func (r *router) handle(p string, t styx.Request) (interface{}, interface{}, error) {
	// ensure the first match based upon path or prefix
	sort.SliceStable(r.handlers, func(i, j int) bool { return r.handlers[i].Path() < r.handlers[j].Path() })
	for _, h := range r.handlers {
		if h.Matches(p) {
			return h.Handler()(strings.TrimPrefix(p, h.Path()), t)
		}
	}
	return nil, nil, ErrNoFile
}
