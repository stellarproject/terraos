package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

const (
	grubPath = "/etc/default/grub"
	grubFile = `GRUB_DEFAULT=0
GRUB_TIMEOUT_STYLE=hidden
GRUB_TIMEOUT=2
GRUB_DISTRIBUTOR="Stellar Project"
GRUB_CMDLINE_LINUX_DEFAULT=""
GRUB_CMDLINE_LINUX="boot=terra vhost=%s"`
)

func mountDevice(device, destination string, action func(string) error) error {
	if device == "" {
		return action(destination)
	}
	tmpMount, err := ioutil.TempDir("", "terra-device-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpMount)
	if err := syscall.Mount(device, tmpMount, "ext4", 0, ""); err != nil {
		return err
	}
	defer syscall.Unmount(tmpMount, 0)
	return action(filepath.Join(tmpMount, destination))
}
