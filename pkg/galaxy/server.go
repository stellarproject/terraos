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
	"fmt"
	"log"
	"os"

	"aqwari.net/net/styx"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/pkg/store"
	"github.com/stellarproject/terraos/pkg/store/client"
	"github.com/stellarproject/terraos/version"
)

// Config is the galaxy server configuration
type Config struct {
	// Addr is the address for the server
	Addr string
	// Debug enables debug logging for the server
	Debug bool
	// StoreURI is the backend data store uri
	StoreURI string
}

// Server is the galaxy server
type Server struct {
	config  *Config
	backend store.Store
	router  *router
}

var logrequests styx.HandlerFunc = func(s *styx.Session) {
	for s.Next() {
		logrus.Debugf("%q %T %s", s.User, s.Request(), s.Request().Path())
	}
}

// NewServer returns a new galaxy server
func NewServer(cfg *Config) (*Server, error) {
	b, err := client.NewStore(cfg.StoreURI)
	if err != nil {
		return nil, err
	}
	r := newRouter()
	r.Path("/", rootHandler)
	r.Path("/version", versionHandler)
	r.Prefix("/cluster", clusterHandler(b))
	return &Server{
		config:  cfg,
		backend: b,
		router:  r,
	}, nil
}

// Run starts a new galaxy server and listens for sessions
func (s *Server) Run() error {
	var styxServer styx.Server
	if s.config.Debug {
		styxServer.ErrorLog = log.New(os.Stderr, "", 0)
		styxServer.TraceLog = log.New(os.Stderr, "", 0)
	}
	styxServer.Addr = s.config.Addr
	styxServer.Handler = styx.Stack(logrequests, s)

	return styxServer.ListenAndServe()
}

func (s *Server) handleRequest(p string) (interface{}, interface{}, error) {
	// top level
	if p == "/" {
		return nil, mkdir([]os.FileInfo{
			&dir{name: "cluster"},
			&file{name: "version"},
		}), nil
	}

	switch p {
	case "/cluster":
		return nil, mkdir([]os.FileInfo{
			&dir{name: "service"},
		}), nil
	case "/cluster/service":
		// TODO: dynamic lookup from redis
		return nil, mkdir([]os.FileInfo{
			&dir{name: "test"},
			&dir{name: "foo"},
		}), nil
	case "/cluster/service/test":
		return nil, mkdir([]os.FileInfo{
			&file{name: "containers"},
		}), nil
	case "/cluster/service/foo":
		return nil, mkdir([]os.FileInfo{
			&file{name: "containers"},
		}), nil
	case "/cluster/service/test/containers":
		return nil, &file{
			name:  "containers",
			isDir: false,
			uid:   "root",
			gid:   "root",
			data:  []byte(`[{id: "test"}, {"id": "foo"}]`),
		}, nil
	case "/cluster/service/foo/containers":
		return nil, &file{
			name:  "containers",
			isDir: false,
			uid:   "root",
			gid:   "root",
			data:  []byte(`[{id: "test"}, {"id": "foo"}]`),
		}, nil
	case "/version":
		return nil, &file{
			name:  "version",
			isDir: false,
			uid:   "root",
			gid:   "root",
			data:  []byte(version.Version + "\n"),
		}, nil
	default:
		// TODO lookup from store
	}
	return nil, nil, ErrNoFile
}

func (s *Server) Serve9P(session *styx.Session) {
	for session.Next() {
		r := session.Request()
		_, f, err := s.router.handle(r.Path(), r)
		if err != nil {
			r.Rerror(err.Error())
			continue
		}
		var fi os.FileInfo
		switch v := f.(type) {
		case *dir:
			fi = v
		case *file:
			fi = v
		default:
			r.Rerror("unknown type received %T", v)
			continue
		}
		switch t := r.(type) {
		case styx.Twalk:
			fmt.Println("Twalk")
			t.Rwalk(fi, nil)
		case styx.Tstat:
			fmt.Println("Tstat")
			fmt.Printf("Tstat fi %T\n", fi)
			t.Rstat(fi, nil)
		case styx.Topen:
			fmt.Println("Topen")
			switch v := fi.(type) {
			case *dir:
				t.Ropen(mkdir(v.entries), nil)
			case *file:
				t.Ropen(fi, nil)
			}
		case styx.Tcreate:
			fmt.Println("Tcreate")
			switch fi.(type) {
			case *dir:
				// TODO: create on the backend
				if t.Mode.IsDir() {
					t.Rcreate(&dir{}, nil)
				} else {
					f, err := s.backend.GetOrCreate(t.Path())
					if err != nil {
						t.Rerror(err.Error())
						continue
					}
					t.Rcreate(f, nil)
				}
			default:
				t.Rerror("%s is not a directory", t.Path())
			}
		default:
			fmt.Printf("default: %T\n", t)
		}
	}
}
