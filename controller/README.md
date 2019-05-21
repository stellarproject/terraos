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
