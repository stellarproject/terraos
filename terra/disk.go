package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var autoPartitionCommand = cli.Command{
	Name:  "auto-partition",
	Usage: "configure the os",
	Action: func(clix *cli.Context) error {
		if err := partition(clix); err != nil {
			return err
		}
		if err := format(clix); err != nil {
			return err
		}
		return installGrub(clix)
	},
}

func partition(clix *cli.Context) error {
	const parted = "o\nn\np\n1\n\n\na\n1\nw"

	cmd := exec.Command("fdisk", clix.GlobalString("device"))
	cmd.Stdin = strings.NewReader(parted)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func format(clix *cli.Context) error {
	cmd := exec.Command(fmt.Sprintf("mkfs.%s", clix.GlobalString("fs-type")), partitionPath(clix.GlobalString("device")))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installGrub(clix *cli.Context) error {
	if err := os.MkdirAll(devicePath, 0755); err != nil {
		return err
	}
	path := partitionPath(clix.GlobalString("device"))
	logrus.WithFields(logrus.Fields{
		"device": path,
		"path":   devicePath,
	}).Info("mounting device")
	if err := syscall.Mount(path, devicePath, clix.GlobalString("fs-type"), 0, ""); err != nil {
		return err
	}
	defer syscall.Unmount(devicePath, 0)

	if err := os.MkdirAll(disk("boot"), 0755); err != nil {
		return err
	}
	if err := syscall.Mount(disk("boot"), "/boot", "none", syscall.MS_BIND, ""); err != nil {
		return err
	}
	defer syscall.Unmount("/boot", 0)
	out, err := exec.Command("grub-install", clix.GlobalString("device")).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
