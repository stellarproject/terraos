# controller

The terra controller provides a management plane for infrastructure.

## dependencies

* tgt
* open-iscsi
* containerd
* orbit
* tftpd-hpa
* btrfs-progs

## Start the controller

```bash
>  sudo terra controller --gateway 10.0.10.1 --iscsi 10.0.10.97
```

## Install the kernel and pxe files

```bash
> terra pxe docker.io/stellarproject/pxe:v10
```

## Create a machine image


To see the current image options:

```bash
> terra create --dump
```

To run a dry run to see the image dockerfile and configuration

```bash
> terra create --dry <server.toml>
```

### Server Image Config

```toml
hostname = "terra-01"
version = "v1"
repo = "docker.io/stellarproject"
os = "docker.io/stellarproject/terraos:v10"
userland = "RUN apt install htop"
init = "/sbin/init"

[[components]]
  image = "docker.io/stellarproject/buildkit:v10"
  systemd = ["buildkit"]

[ssh]
  github = "crosbymichael"

[netplan]

  [[netplan.interfaces]]
    name = "eth0"
    addresses = ["192.168.1.10"]
    gateway = "192.168.1.1"

[resolvconf]
  nameservers = ["8.8.8.8", "8.8.4.4"]
  search = ""
```

```bash
> terra create <server.toml>
```

## Provision a machine

To see the current provision options:

```bash
> terra provision --dump
```

### Provision Config

```toml
hostname = "terra-01"
mac = "xx:xx:xx:xx:xx:xx"
image = "docker.io/stellarproject/example:5"
fs_uri = "iscsi://btrfs"
fs_size = 512

[[fs_subvolumes]]
  name = "tftp"
  path = "/tftp"
```

```bash
> terra provision <node.toml>
```

## Get node information


