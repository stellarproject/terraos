# Terra OS

![terra](iso/splash.png)

Modern, minimal operating system (we've heard that before) optimized for containers within the Stellar Project.

## Install

To install terra on a fresh system make sure you partition and format your disk.
By default, terra will handled `ext4` but you can change the filesystem type with the `--fs-type` flag during install.

Make a bootable USB drive or mount the `terra.iso` to your system and at boot, drop into the live shell.
You can install multiple terra os versions, the current default version to get your system going is `2`.

```bash
> terra install --boot 2
```

Now reboot the system, keep the usb drive in the system to manage it after.  It allows a 2 stage boot from the iso.

You are good to go with terra now.  Have fun.

## Filesystem Layout

Terra supports booting from a single partition disk and will layout the terra files as follows:

```
.
├── boot
│   ├── System.map-5.0.7-terra
│   ├── config-5.0.7-terra
│   ├── grub
│   ├── initrd.img-5.0.7-terra
│   └── vmlinuz-5.0.7-terra
├── config
│   ├── etc
│   └── home
├── odisk
├── os
│   └── 2
├── tmp
├── userdata
│   ├── etc
│   ├── mnt
│   ├── opt
│   ├── root
│   ├── tmp
│   ├── usr
│   └── var
└── work
    └── work
```

* work - overlay working directory
* userdata - `rw` user data of the system
* boot - grub, initrd, and kernel
* odisk - filesystem mounting options
* os - terra os installs
* config - user config layer

## Kernel

We use the latest stable kernel with a stripped down config.
The config is optimized for the KSPP guidelines.

### Kernel Patches

* wireguard

## OS

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
