# Terra OS

![terra](iso/splash.png)

Modern, minimal operating system (we've heard that before) optimized for containers within the Stellar Project.

## Install

Format a new partition and format it.  You *MUST* have the `terra` label applied on your partition that will host terra.

```bash
> mkfs.ext4 -L terra /dev/sda1
```

Make a bootable USB drive or mount the `iso` to your system and at boot, drop into the live shell.
You can install multiple terra os versions.

```bash
> terra --device /dev/sda1 install docker.io/stellarproject/terraos:v5
```

Now reboot the system, keep the usb drive in the system to manage it after.

You are good to go with terra now.  Have fun.

## OS Customizations

You can customize your install by building images based on the released terra os version.


```Dockerfile
FROM docker.io/stellarproject/terraos:v5

RUN echo "testing" > /etc/hostname
```

## Filesystem Layout

Terra supports booting from a single partition disk and will layout the terra files as follows:

```
├── boot
│   ├── System.map-5.0.7-terra
│   ├── config-5.0.7-terra
│   ├── grub
│   ├── initrd.img-5.0.7-terra
│   └── vmlinuz-5.0.7-terra
├── content
│   ├── blobs
│   └── ingest
├── lost+found
├── odisk
├── ov
│   ├── metadata.db
│   └── snapshots
├── tmp
└── vlc
    ├── io.containerd.content.v1.content
    ├── io.containerd.metadata.v1.bolt
    ├── io.containerd.runtime.v1.linux
    ├── io.containerd.runtime.v2.task
    ├── io.containerd.snapshotter.v1.native
    ├── io.containerd.snapshotter.v1.overlayfs
    ├── stellarproject.io.containerd.store.stellarproject.io
    └── tmpmounts

```

* boot - grub, initrd, and kernel
* content - content store
* odisk - filesystem mounting options
* ov - overlay snapshotter
* tmp - tmp directory
* vlc = `/var/lib/containerd` mount point for overlay mounts

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
