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


## Build Terra

If you don't have a kernel built then do so first:

```bash
> make kernel
```

Now build the terra base images and components:

```bash
> make release
```

To make the terra binaries locally and install them do:

```bash
> make local && sudo make install
```

Now that you have the binaries locally, you can start on building your machine images.

## Build a machine image

To create a new server image, create a toml file with an id and any other information needed.
The common name for this file is `server.toml`.

```toml
hostname = "reactor"
labels = ["vm"]
cpus = 12.0
memory = 32768
domain = "compute"

[pxe]
	target = "10.0.10.10"

[network]
	interfaces = ""
	nameservers = ["10.0.10.1"]
	gateway = "10.0.10.1"
	[network.pxe]
		mac = "6c:b3:11:1c:06:9e"
		address = "none"
		bond = ["enp3s0f0", "enp3s0f1"]

[[volumes]]
  label = "os"
  type = "ext4"
  path = "/"
  target_iqn = "iqn.2019.com.crosbymichael.core:reactor"

[image]
	name = "registry.compute:5000/reactor:v4"
	base = "registry.compute:5000/terraos:v16"
	init = "/sbin/init"
	packages = [
		"nfs-utils",
		"libvirt-daemon",
		"qemu-img",
		"qemu-system-x86_64",
		"dbus",
		"polkit",
		"virt-manager"
	]
	services = [
		"dbus",
		"libvirtd",
	]
	userland = """
ADD orbit /etc/init.d/
RUN addgroup terra libvirt
RUN mkdir -p /etc/polkit-1/localauthority/50-local.d
ADD libvirt.pkla /etc/polkit-1/localauthority/50-local.d/50-libvirt-ssh-remote-access-policy.pkla
"""
	[image.ssh]
		github = "https://github.com/crosbymichael.keys"

```

To build the image type:

```bash
> terra create --push server.toml
```

### Test your image

If you have `orbit` installed on your system, you can run your entire server image as a
container and test the install.

You can create a vhost file when you build the image with:

```bash
> terra create --push --vhost vhost.toml server.toml
```

To run the image as a container type:

```bash
> ob create vhost.toml
```

You are now able to exec or ssh into that container by its ip.

## Installation

### PXE

For running in a PXE environment terra has a controller with `tftp` and `iscsi` support.

To install the files for PXE run:

```bash
> terra pxe install registry.compute:5000/pxe:v16-dev
```

To save your node image in pxe run:

```bash
> terra pxe save server.toml
```

This will generate the pxe config for the node's information.

### Disk

Make sure you have a disk partitioned with one partition when installing terra.
You need to have this done before using the terra command to install the image to the device.
Use `fdisk` here to do so.

To install your image onto a disk, run the following.

To install onto an iscsi volume:

```bash
> terra install server.toml
```

To select the physical or already mounted device run:

```bash
> terra  install --device os:/dev/sda1
```

## Boot

Now boot your node and PXE will take care of the rest.

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
