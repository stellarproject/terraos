# Terra OS

![terra](iso/splash.png)

Modern, minimal operating system (we've heard that before) optimized for containers within the Stellar Project.

## Build

To build the iso and pxe targets just type `make`.

## Install

Format a new partition on your disk of choice.
You *MUST* have the `terra` label applied on your partition that will host terra.

```bash
> mkfs.ext4 -L terra /dev/sda1
```

```bash
> terra --device /dev/sda1 install <image>
```

Now reboot the system, keep the usb drive in the system to manage it after.

You are good to go with terra now.  Have fun.

## OS Customizations

You can customize your install by building images based on the released terra os version.

```toml
id = "cm-02"
version = "v1"
repo = "docker.io/crosbymichael"
os = "docker.io/stellarproject/terraos:v7"

[ssh]
	github = "crosbymichael"

[netplan]
	interface = "eno1"

userland = "ADD config.toml /etc/containerd/"

[pxe]
	mac = "xx:xx:xx:xx:xx:xx"
	target_ip = "xxx.xxx.x.xx"
	target = "xx"
	iqn = "iqn.2019-01.xxx.com"
```

```bash
> terra create server.toml
```

If `[pxe]` is specified, it will output a pxe specific config for the boot server for this image.
It will setup iscsi targets correctly.

## Kernel

We use the latest stable kernel with a stripped down config.
The config is optimized for the KSPP guidelines.

### Kernel Patches

* wireguard

### Services

* containerd - runtime
	* orbit
	* stellar store
	* stellar dns
* buildkit - image builder
* Prometheus Node Exporter - node metrics

### Binaries

* criu - checkpoint and restore
* cni plugins - networking
* vab - image build frontend
* terra - post install management

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
