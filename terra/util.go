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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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
	terraRepoFormat = "docker.io/stellarproject/terraos:%d"
	bootRepoFormat  = "docker.io/stellarproject/boot:%d"
	devicePath      = "/run/terramnt"
)

var defaultMountOptions = []string{
	"lowerdir=/lower/config:/lower/os/2",
	"upperdir=/lower/userdata",
	"workdir=/lower/work",
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

func getVersion(clix *cli.Context) (int, error) {
	version := clix.Args().First()
	if version == "" {
		return 0, errNoOS
	}
	return strconv.Atoi(version)
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

func applyImage(clix *cli.Context, cs content.Store, imageName, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	authorizer := docker.NewAuthorizer(nil, getDockerCredentials)
	resolver := docker.NewResolver(docker.ResolverOptions{
		PlainHTTP:  clix.Bool("http"),
		Authorizer: authorizer,
	})
	ctx := context.Background()
	name, desc, err := resolver.Resolve(ctx, imageName)
	if err != nil {
		return err
	}
	fetcher, err := resolver.Fetcher(ctx, name)
	if err != nil {
		return err
	}
	logrus.Infof("fetching os %s", imageName)
	childrenHandler := images.ChildrenHandler(cs)
	h := images.Handlers(remotes.FetchHandler(cs, fetcher), childrenHandler)
	if err := images.Dispatch(ctx, h, nil, desc); err != nil {
		return err
	}
	_, layers, err := getLayers(ctx, cs, desc)
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

// RegistryAuth is the base64 encoded credentials for the registry credentials
type RegistryAuth struct {
	Auth string `json:"auth,omitempty"`
}

// DockerConfig is the docker config struct
type DockerConfig struct {
	Auths map[string]RegistryAuth `json:"auths"`
}

func getDockerCredentials(host string) (string, string, error) {
	logrus.WithField("host", host).Debug("checking for registry auth config")
	home := os.Getenv("HOME")
	credentialConfig := filepath.Join(home, ".docker", "config.json")
	f, err := os.Open(credentialConfig)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}
		return "", "", err
	}
	defer f.Close()

	var cfg DockerConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return "", "", err
	}

	for h, r := range cfg.Auths {
		if h == host {
			creds, err := base64.StdEncoding.DecodeString(r.Auth)
			if err != nil {
				return "", "", err
			}
			parts := strings.SplitN(string(creds), ":", 2)
			logrus.Debugf("using auth for registry %s: user=%s", host, parts[0])
			return parts[0], parts[1], nil
		}
	}

	return "", "", nil
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
