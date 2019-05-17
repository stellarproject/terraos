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

package v1

import "encoding/json"

const cniVersion = "0.3.1"

type cni struct {
	Version          string `json:"cniVersion,omitempty"`
	Name             string `json:"name,omitempty"`
	Type             string `json:"type,omitempty"`
	Master           string `json:"master,omitempty"`
	IPAM             ipam   `json:"ipam,omitempty"`
	Bridge           string `json:"bridge,omitempty"`
	IsDefaultGateway bool   `json:"isDefaultGateway,omitempty"`
	IPMask           bool   `json:"ipMasq,omitempty"`
}

type ipam struct {
	Type        string `json:"type,omitempty"`
	Subnet      string `json:"subnet,omitempty"`
	SubnetRange string `json:"subnet_range,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
}

func (n *CNINetwork) MarshalCNI() []byte {
	c := cni{
		Version: cniVersion,
		Name:    n.Name,
		Type:    n.Type,
		Master:  n.Master,
		Bridge:  n.Bridge,
	}
	if n.Type == "bridge" {
		c.IsDefaultGateway = true
		c.IPMask = true
	}
	if n.IPAM != nil {
		c.IPAM.Type = n.IPAM.Type
		c.IPAM.Subnet = n.IPAM.Subnet
		c.IPAM.SubnetRange = n.IPAM.SubnetRange
		c.IPAM.Gateway = n.IPAM.Gateway
	}
	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return data
}
