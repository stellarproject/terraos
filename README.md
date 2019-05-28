# Terra OS

![terra](iso/splash.png)

Modern, minimal operating system (we've heard that before) optimized for containers within the Stellar Project.

Terra is an OS not a distro, we use the best distrobution for the job, and that's an Ubuntu base.

This repo is a mono repo for most, if not all terra && stellar projects.

## Status

**Project Status:** alpha

* alpha - APIs will change, not ready services.
* beta - APIs may change, can run services.
* production - APIs stable, can run services.

### Component Status

* `terra`: alpha
* `terra controller`: alpha
* `terra-install`: beta
* `stage0`: alpha
* `stage1`: alpha
* `iso`: alpha
* `orbit`: alpha
* `vab`: beta

## Build a machine image

To create a new server image, create a toml file with an id and any other information needed.
The common name for this file is `server.toml`.

```toml
hostname = "example"
version = "6"
os = "docker.io/stellarproject/terraos:v11"
repo = "docker.io/stellarproject"
userland = """
RUN echo 'terra:terra' | chpasswd
ADD sshd_config /etc/ssh/
"""

[netplan]
  [[netplan.interfaces]]
    name = "eth0"

[resolvconf]
  nameservers = ["8.8.8.8", "8.8.4.4"]
```

To build the image type:

```bash
> terra create --push server.toml
```

### Test your image

If you have `orbit` installed on your system, you can run your entire server image as a
container and test the install.

Create a vhost.toml file for orbit.

```toml
id = "terra-vhost"
image = "docker.io/stellarproject/example:6"

uid = 0
gid = 0
privileged = true
# systemd will try to setup the interface
masked_paths = ["/etc/netplan"]

[[networks]]
	type = "macvlan"
	name = "ob0"
	[networks.ipam]
		type = "dhcp"

[resources]
	cpu = 1.0
	memory = 128
```

To run the image as a container type:

```bash
> ob create vhost.toml
```

You are now able to exec or ssh into that container by its ip.

## Installation

To install a server image on a machine you need to create a `node.toml` file.
This file specifies the image, mac, hostname, and disk groups.
Terra supports `btrfs` out of the box and sets up subvolumes as well as raid support.

**Simple Node:**

```toml
hostname = "terra-01"
image = "docker.io/stellarproject/example:6"

[[groups]]
  label = "os"
  type = "single"
  stage = "stage1"
  mbr = true

  [[groups.disk]]
    device = "/dev/sda"
```

**Complex Node:**

```toml
hostname = "mt-01"
image = "docker.io/crosbymichael/mt-01:v2"

[[groups]]
  label = "os"
  type = "single"
  stage = "stage1"
  mbr = true

  [[groups.disk]]
    device = "/dev/nvme0n1p1"

  [[groups.subvolumes]]
    name = "home"
    path = "/home"

  [[groups.subvolumes]]
    name = "log"
    path = "/var/log"

  [[groups.subvolumes]]
    name = "containerd"
    path = "/var/lib/containerd"


[[groups]]
  label = "storage"
  type = "raid10"
  stage = "stage1"

  [[groups.subvolumes]]
    name = "luns"
    path = "/iscsi"

  [[groups.subvolumes]]
    name = "tftp"
    path = "/tftp"

  [[groups.disk]]
    device = "/dev/sda"

  [[groups.disk]]
    device = "/dev/sdb"

  [[groups.disk]]
    device = "/dev/sdd"

  [[groups.disk]]
    device = "/dev/sde"
```

### Bare Metal

To install your server image on a bare metal machine, write the `terra.iso` to a USB and boot into the live setup cd.

Either host your `node.toml` or scp it to the live machine over the builtin ssh server when it boots.

```bash
> terra-install --gateway <yourgatewayip> node.toml
```

```bash
> terra-install --gateway <yourgatewayip> https://nodes/node.toml
```

### PXE

For running in a PXE environment terra has a controller with `tftp` and `iscsi` support.

After you get a controller running you need to install the `pxe` image with the kernel and boot code.

#### Controller

`controller.service`:

```
[Unit]
Description=terra controller
After=containerd.service orbit.service network.target

[Service]
ExecStart=/usr/local/sbin/terra --debug --controller 10.0.10.3 controller --gateway 10.0.10.1 --iscsi 10.0.10.4
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

For pxe booting, you will need to add a mac and disk size to your `node.toml`.
Disk sizes are in MB.

```toml
hostname = "terra-01"
mac = "xx:xx:xx:xx:xx:xx"
image = "docker.io/stellarproject/example:6"

[[groups]]
  label = "os"
  type = "single"
  stage = "stage1"

  [[groups.disk]]
    device = "/dev/sda"
	size = 128000 # 128G
```

```bash
> terra pxe docker.io/stellarproject/pxe:v11
```

Then provision the node with the `provision` command.

```bash
> terra provision node.toml
```

You will get output like this when the node is provisioned and ready to be booted:

```json
{
 "hostname": "cm-01",
 "mac": "54:b2:03:07:f1:d1",
 "image": "docker.io/crosbymichael/cm-01:v5",
 "disk_groups": [
  {
   "label": "os",
   "stage": 1,
   "disks": [
    {
     "id": 1,
     "device": "/iscsi/cm-01/0.lun",
     "fs_size": 128000
    }
   ],
   "subvolumes": [
    {
     "name": "home",
     "path": "/home"
    },
    {
     "name": "log",
     "path": "/var/log"
    },
    {
     "name": "containerd",
     "path": "/var/lib/containerd"
    }
   ],
   "target": {
    "iqn": "iqn.2024.san.crosbymichael.com.cm-01:fs",
    "id": 2
   }
  }
 ],
 "initiator_iqn": "iqn.2024.node.crosbymichael.com:cm-01"
}
```


You can manage and see nodes with the `terra` command:

```bash
> terra list

HOSTNAME   MAC                 IMAGE                              INITIATOR                               TARGET
cm-01      54:b2:03:07:f1:d1   docker.io/crosbymichael/cm-01:v5   iqn.2024.node.crosbymichael.com:cm-01   iqn.2024.san.crosbymichael.com.cm-01:fs
```

## Build Release

To build all the terra images type:

```bash

make release

```

## License

```
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
```
