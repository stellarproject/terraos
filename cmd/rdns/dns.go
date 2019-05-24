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
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type Key struct {
	Domain string
	Name   string
	ID     string
}

func (k Key) isEmpty() bool {
	return k.Domain == ""
}

func getKey(q dns.Question) Key {
	parts := strings.Split(q.Name, ".")
	switch len(parts) {
	case 3:
		return Key{
			Domain: parts[1],
			Name:   parts[0],
		}
	case 4:
		return Key{
			Domain: parts[2],
			Name:   parts[1],
			ID:     parts[0],
		}
	}
	return Key{}
}

func createHeader(rtype uint16, name string, ttl uint32) dns.RR_Header {
	return dns.RR_Header{
		Name:   name,
		Rrtype: rtype,
		Class:  dns.ClassINET,
		Ttl:    ttl,
	}
}

func getProto(w dns.ResponseWriter) string {
	proto := "udp"
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		proto = "tcp"
	}
	return proto
}

func startDnsServer(mux *dns.ServeMux, net, addr string, udpsize int, timeout time.Duration) error {
	s := &dns.Server{
		Addr:         addr,
		Net:          net,
		Handler:      mux,
		UDPSize:      udpsize,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}
	return s.ListenAndServe()
}
