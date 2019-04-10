/*
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
*/

package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd/archive"
	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/rootfs"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	terraRepoFormat = "docker.io/stellarproject/terraos:%s"
	bootRepoFormat  = "docker.io/stellarproject/boot:%s"
	devicePath      = "/run/terramnt"
)

var defaultMountOptions = []string{
	"lowerdir=/sd/config:/sd/os/2",
	"upperdir=/sd/userdata",
	"workdir=/sd/work",
}

var (
	errNoOS = errors.New("no os version specified")
	errNoID = errors.New("no id specified")
)

func disk(args ...string) string {
	return filepath.Join(append([]string{devicePath}, args...)...)
}

func partitionPath(clix *cli.Context) string {
	return fmt.Sprintf("%s%d", clix.GlobalString("device"), clix.GlobalInt("partition"))
}

// before mounts the device before doing operations
func before(clix *cli.Context) error {
	if err := os.MkdirAll(devicePath, 0755); err != nil {
		return err
	}
	return syscall.Mount(partitionPath(clix), devicePath, clix.GlobalString("fs-type"), 0, "")
}

// after unmounts the device
func after(clix *cli.Context) error {
	for i := 0; i < 30; i++ {
		if err := syscall.Unmount(devicePath, 0); err == nil {
			return err
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("unable to remove mount")
}

func getVersion(clix *cli.Context) (string, error) {
	version := clix.Args().First()
	if version == "" {
		return "", errNoOS
	}
	return version, nil
}

func newContentStore() (content.Store, error) {
	if err := os.MkdirAll(disk("/tmp/content"), 0755); err != nil {
		return nil, err
	}
	tmpContent, err := ioutil.TempDir(disk("/tmp/content"), "terra-content-")
	if err != nil {
		return nil, err
	}
	return local.NewStore(tmpContent)
}

func fetch(ctx context.Context, clix *cli.Context, cs content.Store, imageName string) (*v1.Descriptor, error) {
	authorizer := docker.NewAuthorizer(nil, getDockerCredentials)
	resolver := docker.NewResolver(docker.ResolverOptions{
		PlainHTTP:  clix.Bool("http"),
		Authorizer: authorizer,
	})
	name, desc, err := resolver.Resolve(ctx, imageName)
	if err != nil {
		return nil, err
	}
	fetcher, err := resolver.Fetcher(ctx, name)
	if err != nil {
		return nil, err
	}
	logrus.Infof("fetching os %s", imageName)
	childrenHandler := images.ChildrenHandler(cs)
	h := images.Handlers(remotes.FetchHandler(cs, fetcher), childrenHandler)
	if err := images.Dispatch(ctx, h, nil, desc); err != nil {
		return nil, err
	}
	return &desc, nil
}

func unpackFlat(ctx context.Context, cs content.Store, desc *v1.Descriptor, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	_, layers, err := getLayers(ctx, cs, *desc)
	if err != nil {
		return err
	}
	logrus.Infof("unpacking os to %q", dest)
	for _, layer := range layers {
		if err := extract(ctx, cs, layer, dest); err != nil {
			return err
		}
	}
	return nil
}

func extract(ctx context.Context, cs content.Store, layer rootfs.Layer, dest string) error {
	ra, err := cs.ReaderAt(ctx, layer.Blob)
	if err != nil {
		return err
	}
	defer ra.Close()

	cr := content.NewReader(ra)
	r, err := compression.DecompressStream(cr)
	if err != nil {
		return err
	}
	defer r.Close()

	if r.(compression.DecompressReadCloser).GetCompression() == compression.Uncompressed {
		return nil
	}
	logrus.WithField("layer", layer.Blob.Digest).Info("apply layer")
	if _, err := archive.Apply(ctx, dest, r, archive.WithFilter(HostFilter)); err != nil {
		return err
	}
	return nil
}

const excludedModes = os.ModeDevice | os.ModeCharDevice | os.ModeSocket | os.ModeNamedPipe

func HostFilter(h *tar.Header) (bool, error) {
	// exclude devices
	if h.FileInfo().Mode()&excludedModes != 0 {
		return false, nil
	}
	return true, nil
}

func getConfig(ctx context.Context, provider content.Provider, desc v1.Descriptor) (*Image, error) {
	p, err := content.ReadBlob(ctx, provider, desc)
	if err != nil {
		return nil, err
	}
	var config Image
	if err := json.Unmarshal(p, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func getLayers(ctx context.Context, cs content.Store, desc v1.Descriptor) (*Image, []rootfs.Layer, error) {
	manifest, err := images.Manifest(ctx, cs, desc, nil)
	if err != nil {
		return nil, nil, err
	}
	config, err := getConfig(ctx, cs, manifest.Config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to resolve config")
	}
	diffIDs := config.RootFS.DiffIDs
	if len(diffIDs) != len(manifest.Layers) {
		return nil, nil, errors.Errorf("mismatched image rootfs and manifest layers")
	}
	var layers []rootfs.Layer
	for i := range diffIDs {
		var l rootfs.Layer
		l.Diff = v1.Descriptor{
			MediaType: v1.MediaTypeImageLayer,
			Digest:    diffIDs[i],
		}
		l.Blob = manifest.Layers[i]
		layers = append(layers, l)
	}
	return config, layers, nil
}

func overlayBoot() (func() error, error) {
	if err := syscall.Mount(disk("boot"), "/boot", "none", syscall.MS_BIND, ""); err != nil {
		return nil, err
	}
	return func() error {
		return syscall.Unmount("/boot", 0)
	}, nil
}

func writeMountOptions(options []string) error {
	f, err := os.OpenFile(disk("odisk"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(strings.Join(options, ","))
	return err
}
