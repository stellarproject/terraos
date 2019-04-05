package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var enableCommand = cli.Command{
	Name:      "enable",
	Usage:     "enable the provided vhost",
	ArgsUsage: "VHOST",
	Action: func(clix *cli.Context) error {
		vhost := clix.Args().First()
		if vhost == "" {
			return errNoVhost
		}
		// make the config dir if it does not exist
		if err := os.MkdirAll(filepath.Join(clix.GlobalString("root"), "config"), 0755); err != nil {
			return err
		}
		return mountDevice(clix.GlobalString("device"), "/", func(dev string) error {
			return enable(clix, vhost, dev)
		})
	},
}

func enable(clix *cli.Context, vhost, dev string) (err error) {
	var (
		hostBoot = filepath.Join(dev, "boot")
		vhostDir = filepath.Join(dev, clix.GlobalString("root"), vhost)
	)
	// make sure we have a boot directory for the vhost
	if _, err := os.Stat(filepath.Join(vhostDir, "boot")); err != nil {
		return errors.Wrapf(err, "boot path does not exist for vhost %s", vhost)
	}
	if err := copyVhostBoot(hostBoot, filepath.Join(vhostDir, "boot")); err != nil {
		return errors.Wrap(err, "change boot to current")
	}
	return updateGrub(vhost)
}

func copyVhostBoot(host, current string) error {
	return filepath.Walk(current, func(path string, info os.FileInfo, err error) error {
		if path == current {
			return nil
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		name := info.Name()
		f, err := os.Create(filepath.Join(host, name))
		if err != nil {
			return err
		}
		c, err := os.Open(path)
		if err != nil {
			return err
		}
		defer c.Close()
		defer f.Close()
		_, err = io.Copy(f, c)
		return err
	})
}

func updateGrub(vhost string) error {
	f, err := ioutil.TempFile("", "terra-grub-")
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf(grubFile, vhost))
	f.Close()
	if err != nil {
		return err
	}
	if err := os.Rename(f.Name(), grubPath); err != nil {
		return err
	}
	cmd := exec.Command("update-grub2")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", out)
	}
	return nil
}
