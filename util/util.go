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

package util

import (
	"errors"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

var (
	ErrIPAddressNotFound = errors.New("ip address for interface not found")
	ErrNoDefaultRoute    = errors.New("no default route found")
)

func GetIP(name string) (string, error) {
	i, err := net.InterfaceByName(name)
	if err != nil {
		return "", err
	}
	return getIPf(i, ipv4)
}

func getIPf(i *net.Interface, ipfunc func(n *net.IPNet) string) (string, error) {
	addrs, err := i.Addrs()
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		s := ipfunc(n)
		if s == "" {
			continue
		}
		return s, nil
	}
	return "", ErrIPAddressNotFound
}

func ipv4(n *net.IPNet) string {
	if n.IP.To4() == nil {
		return ""
	}
	return n.IP.To4().String()
}

func GetDomainName() (name string, err error) {
	data, err := ioutil.ReadFile("/proc/sys/kernel/domainname")
	if err != nil {
		return "", err
	}
	s := strings.TrimRight(string(data), "\n")
	if s == "(none)" {
		return "local", nil
	}
	return s, nil
}

func GetDefaultIface() (string, error) {
	for i := 0; i < 15; i++ {
		routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
		if err != nil {
			return "", err
		}
		for _, r := range routes {
			if r.Gw != nil {
				link, err := netlink.LinkByIndex(r.LinkIndex)
				if err != nil {
					return "", err
				}
				return link.Attrs().Name, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", ErrNoDefaultRoute
}

func IPAndGateway() (string, string, error) {
	for i := 0; i < 15; i++ {
		routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
		if err != nil {
			return "", "", err
		}
		for _, r := range routes {
			if r.Gw != nil {
				link, err := netlink.LinkByIndex(r.LinkIndex)
				if err != nil {
					return "", "", err
				}
				name := link.Attrs().Name
				ip, err := GetIP(name)
				if err != nil {
					return "", "", err
				}
				return ip, r.Gw.To4().String(), nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", "", ErrNoDefaultRoute
}
