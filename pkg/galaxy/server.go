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
	path    string
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
	r.Path("/version", versionHandler(b))
	return &Server{
		config:  cfg,
		backend: b,
		router:  r,
		path:    "/tmp/stellar-store",
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
		switch t := r.(type) {
		case styx.Twalk:
			fmt.Println("Twalk", r.Path())
			info, err := os.Lstat(filepath.Join(s.path, r.Path()))
			if err != nil {
				if os.IsNotExist(err) {
					t.Rwalk(nil, os.ErrNotExist)
					continue
				}
				t.Rwalk(nil, err)
				continue
			}
			t.Rwalk(newFileInfo(info), err)
		case styx.Tstat:
			fmt.Println("Tstat", r.Path())
			info, err := os.Stat(filepath.Join(s.path, r.Path()))
			if err != nil {
				if os.IsNotExist(err) {
					t.Rstat(nil, os.ErrNotExist)
					continue
				}
				t.Rstat(nil, err)
			}
			t.Rstat(newFileInfo(info), err)
		case styx.Tchown:
			path := filepath.Join(s.path, t.Path())
			err := os.Chown(path, t.Uid, t.Gid)
			t.Rchown(err)
		case styx.Tchmod:
			path := filepath.Join(s.path, t.Path())
			err := os.Chmod(path, t.Mode)
			t.Rchmod(err)
		case styx.Trename:

		case styx.Tsync:
			f, err := s.open(t.Path())
			if err != nil {
				t.Rsync(err)
				continue
			}
			t.Rsync(f.Sync())
			f.Close()
		case styx.Tutimes:
			fmt.Println("Tutimes", r.Path())
			// TODO
			t.Rutimes(nil)
		case styx.Topen:
			fmt.Println("Topen", r.Path())
			f, err := s.open(r.Path())
			t.Ropen(f, err)
		case styx.Ttruncate:
			f, err := s.open(r.Path())
			if err != nil {
				t.Rtruncate(err)
				continue
			}
			t.Rtruncate(f.Truncate(t.Size))
		case styx.Tcreate:
			path := filepath.Join(s.path, t.NewPath())
			if t.Mode&os.ModeDir == os.ModeDir {
				if err := os.Mkdir(path, t.Mode); err != nil {
					t.Rcreate(nil, err)
					continue
				}
				f, err := s.open(t.NewPath())
				if err != nil {
					t.Rcreate(nil, err)
					continue
				}
				t.Rcreate(f, nil)
			} else {
				t.Rcreate(os.OpenFile(path, os.O_CREATE|t.Flag, t.Mode))
			}
		case styx.Tremove:
			//			t.Rremove(nil)
		default:
			fmt.Printf("default: %T\n", t)
		}
	}
}

func (s *Server) open(path string) (*os.File, error) {
	f, err := os.Open(filepath.Join(s.path, path))
	if err != nil && os.IsNotExist(err) {
		err = os.ErrNotExist
	}
	return f, err
}

func newFileInfo(info os.FileInfo) os.FileInfo {
	return info
	/*
		stat := info.Sys().(*syscall.Stat_t)
		return &fileInfo{
			FileInfo: info,
			uid:      fmt.Sprintf("%d", stat.Uid),
			gid:      fmt.Sprintf("%d", stat.Gid),
		}
	*/
}

type fileInfo struct {
	os.FileInfo
	uid string
	gid string
}

func (i *fileInfo) Uid() string {
	fmt.Println("called Uid()", i.uid)
	return i.uid
}

func (i *fileInfo) Gid() string {
	return i.gid
}
