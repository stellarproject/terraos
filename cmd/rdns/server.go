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
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gomodule/redigo/redis"
	mdns "github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

func New(pool *redis.Pool) *Server {
	return &Server{
		IP:      "0.0.0.0",
		Port:    53,
		UDPSize: 65535,
		Timeout: 2 * time.Second,
		TTL:     300,
		Nameservers: []string{
			"8.8.8.8:53",
			"8.8.4.4:53",
		},
		pool: pool,
	}
}

type Server struct {
	IP          string
	Port        int
	UDPSize     int
	Timeout     time.Duration
	TTL         uint32
	Nameservers []string
	Domain      string

	pool *redis.Pool
}

func (s *Server) Serve(ctx context.Context) error {
	mux := mdns.NewServeMux()
	mux.Handle(".", s)

	address := fmt.Sprintf("%s:%d", s.IP, s.Port)

	errCh := make(chan error, 2)

	go func() {
		errCh <- startDnsServer(mux, "tcp", address, 0, s.Timeout)
	}()
	go func() {
		errCh <- startDnsServer(mux, "udp", address, s.UDPSize, s.Timeout)
	}()
	select {
	case err := <-errCh:
		s.pool.Close()
		return err
	case <-ctx.Done():
		s.pool.Close()
		return ctx.Err()
	}
}

func (s *Server) ServeDNS(w mdns.ResponseWriter, req *mdns.Msg) {
	question := req.Question[0]
	key := getKey(question)
	if key.Domain != s.Domain {
		s.ServeDNSForward(w, req)
		return
	}

	m := &mdns.Msg{
		MsgHdr: mdns.MsgHdr{
			Authoritative:      true,
			RecursionAvailable: true,
		},
	}

	m.SetReply(req)
	services, err := fetchServices(s.pool, key.Name)
	if err != nil {
		logrus.WithError(err).Error("dns: fetch service")
		m.SetRcode(req, mdns.RcodeNameError)
		return
	}

	if key.isEmpty() {
		m.SetRcode(req, mdns.RcodeNameError)
		return
	}
	target := fmt.Sprintf("%s.%s.", key.Name, key.Domain)
	for _, srv := range services {
		ip := net.ParseIP(srv.IP)

		switch question.Qtype {
		case mdns.TypeA, mdns.TypeAAAA, mdns.TypeANY:
			m.Answer = append(m.Answer, &mdns.A{
				Hdr: createHeader(mdns.TypeA, question.Name, s.TTL),
				A:   ip,
			})
		case mdns.TypeSRV:
			m.Answer = append(m.Answer, &mdns.SRV{
				Hdr:    createHeader(mdns.TypeSRV, formatSRV(target, srv.Name, "tcp"), s.TTL),
				Port:   uint16(srv.Port),
				Target: target,
			})
		default:
			m.SetRcode(req, mdns.RcodeNameError)
		}
	}
	if err := w.WriteMsg(m); err != nil {
		logrus.WithError(err).Error("dns: write message")
	}
}

func (s *Server) ServeDNSForward(w mdns.ResponseWriter, req *mdns.Msg) {
	var (
		try    = 0
		client = &mdns.Client{
			Net:         getProto(w),
			ReadTimeout: s.Timeout,
		}
		nameservers = append([]string{}, s.Nameservers...)
		nsid        = int(req.Id) % len(nameservers)
	)
	// Use request Id for "random" nameserver selection
	for try < len(nameservers) {
		// TODO: we can cache this in redis with a ttl
		r, _, err := client.Exchange(req, nameservers[nsid])
		if err == nil {
			w.WriteMsg(r)
			return
		}
		logrus.WithError(err).Errorf("dns: exchange %s", nameservers[nsid])
		// Seen an error, this can only mean, "DNSServer not reached", try again
		// but only if we have not exausted our nameservers
		try++
		nsid = (nsid + 1) % len(nameservers)
	}

	m := &mdns.Msg{}
	m.SetReply(req)
	m.SetRcode(req, mdns.RcodeServerFailure)

	if err := w.WriteMsg(m); err != nil {
		logrus.WithError(err).Error("dns: write message")
	}
}

func formatSRV(target, name, protocol string) string {
	return fmt.Sprintf("_%s._%s.%s", name, protocol, target)
}
