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
	"path/filepath"

	"aqwari.net/net/styx"
	"github.com/sirupsen/logrus"
	"github.com/stellarproject/terraos/pkg/store"
	"github.com/stellarproject/terraos/pkg/store/client"
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
	router  *Router
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
	r := NewRouter()
	r.Path("/", rootHandler(b))
	r.Path("/version", versionHandler(b))
	r.Prefix("/cluster", clusterHandler(b))
	return &Server{
		config:  cfg,
		backend: b,
		router:  r,
	}, nil
}

// Router returns the current router for the server
func (s *Server) Router() *Router {
	return s.router
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

// Serve9P implements the 9P session request handler
func (s *Server) Serve9P(session *styx.Session) {
	for session.Next() {
		r := session.Request()
		_, f, err := s.router.handle(r.Path(), r)
		if err != nil {
			//r.Rerror(err.Error())
			//continue
		}
		var fi os.FileInfo
		switch v := f.(type) {
		case Dir:
			fi = v
		case File:
			fi = v
		case nil:
			// create
			fi = nil
		default:
			r.Rerror("unknown type received %T", v)
			continue
		}

		switch t := r.(type) {
		case styx.Twalk:
			fmt.Println("Twalk")
			if fi != nil {
				t.Rwalk(fi, nil)
			} else {
				t.Rwalk(nil, os.ErrNotExist)
			}
		case styx.Tstat:
			fmt.Println("Tstat")
			fmt.Printf("Tstat fi %T\n", fi)
			if fi != nil {
				t.Rstat(fi, nil)
			} else { // attempt to create
				if _, err := s.backend.GetOrCreate(t.Path()); err != nil {
					t.Rerror(err.Error())
					continue
				}
				n := filepath.Base(t.Path())
				f, err := newFile(n, t.Path(), "0", "0", false, nil, s.backend)
				if err != nil {
					t.Rerror(err.Error())
					continue
				}
				t.Rstat(f, nil)
			}
		case styx.Tutimes:
			fmt.Println("Tutimes")
			// TODO
			t.Rutimes(nil)
		case styx.Topen:
			fmt.Println("Topen")
			switch v := fi.(type) {
			case *dir:
				t.Ropen(mkdir(v.entries, 0755), nil)
			case *file:
				t.Ropen(fi, nil)
			}
		case styx.Ttruncate:
			// TODO
			t.Rtruncate(nil)
		case styx.Tcreate:
			fmt.Println("Tcreate")
			switch fi.(type) {
			case *dir:
				// TODO: create on the backend
				if t.Mode.IsDir() {
					t.Rcreate(&dir{}, nil)
				} else {
					f, err := s.backend.GetOrCreate(t.Name)
					if err != nil {
						t.Rerror(err.Error())
						continue
					}
					t.Rcreate(f, nil)
				}
			default:
				t.Rerror("%s is not a directory", t.Path())
			}
		case styx.Tremove:
			if err := s.backend.Delete(t.Path()); err != nil {
				t.Rerror(err.Error())
				continue
			}
			t.Rremove(nil)
		default:
			fmt.Printf("default: %T\n", t)
		}
	}
}
