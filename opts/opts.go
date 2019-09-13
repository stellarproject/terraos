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
	HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
	TORT OR OTHERWISE,
	ARISING FROM, OUT OF OR IN CONNECTION WITH
	THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package opts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd"
	api "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/contrib/apparmor"
	"github.com/containerd/containerd/contrib/nvidia"
	"github.com/containerd/containerd/contrib/seccomp"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/typeurl"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	is "github.com/opencontainers/image-spec/specs-go/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	v1 "github.com/stellarproject/terraos/api/v1/orbit"
	"github.com/stellarproject/terraos/pkg/iscsi"
)

const (
	CurrentConfig          = "stellarproject.io/orbit/container"
	LastConfig             = "stellarproject.io/orbit/container.last"
	IPLabel                = "stellarproject.io/orbit/container.ip"
	RestoreCheckpointLabel = "stellarproject.io/orbit/restore.checkpoint"
)

type Paths struct {
	State   string
	Cluster string
}

func (p Paths) NetworkPath(id string) string {
	return filepath.Join(p.State, "net")
}

func (p Paths) ConfigPath(name string) string {
	return filepath.Join(p.Cluster, "configs", name)
}

// WithOrbitConfig is a containerd.NewContainerOpts for spec and container configuration
func WithOrbitConfig(paths Paths, config *v1.Container, image containerd.Image) func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
	return func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
		// generate the spec
		if err := containerd.WithNewSpec(specOpt(paths, config, image))(ctx, client, c); err != nil {
			return err
		}
		// save the config as a container extension
		return containerd.WithContainerExtension(CurrentConfig, config)(ctx, client, c)
	}
}

func WithSetPreviousConfig(ctx context.Context, client *containerd.Client, c *containers.Container) error {
	c.Extensions[LastConfig] = c.Extensions[CurrentConfig]
	return nil
}

func WithRollback(ctx context.Context, client *containerd.Client, c *containers.Container) error {
	d := c.Extensions[LastConfig]
	if d.Value == nil {
		return nil
	}
	c.Extensions[CurrentConfig] = d
	return nil
}

func specOpt(paths Paths, container *v1.Container, image containerd.Image) oci.SpecOpts {
	opts := []oci.SpecOpts{
		oci.WithImageConfigArgs(image, container.Process.Args),
		oci.WithHostLocaltime,
		oci.WithEnv(container.Process.Env),
		withMounts(container.Mounts),
		withConfigs(paths, container.Configs),
		oci.WithHostname(container.ID),
	}

	if container.Security.Privileged {
		opts = append(opts,
			oci.WithPrivileged,
			oci.WithNewPrivileges,
		)
	} else {
		opts = append(opts,
			oci.WithNoNewPrivileges,
			apparmor.WithDefaultProfile("orbit"),
			seccomp.WithDefaultProfile(),
		)
	}
	if len(container.Security.MaskedPaths) > 0 {
		opts = append(opts, oci.WithMaskedPaths(container.Security.MaskedPaths))
	}
	if container.Process.Pty {
		opts = append(opts, oci.WithTTY)
	}
	if len(container.Networks) == 1 &&
		container.Networks[0].TypeUrl == proto.MessageName(&v1.HostNetwork{}) {
		opts = append(opts, oci.WithHostHostsFile, oci.WithHostResolvconf, oci.WithHostNamespace(specs.NetworkNamespace))
	} else {
		opts = append(opts, oci.WithHostResolvconf, WithContainerHostsFile(paths.State), oci.WithLinuxNamespace(specs.LinuxNamespace{
			Type: specs.NetworkNamespace,
			Path: paths.NetworkPath(container.ID),
		}),
		)
	}
	if container.Resources != nil {
		opts = append(opts, withResources(container.Resources))
	}
	if container.Gpus != nil {
		opts = append(opts, nvidia.WithGPUs(
			nvidia.WithDevices(ints(container.Gpus.Devices)...),
			nvidia.WithCapabilities(toGpuCaps(container.Gpus.Capabilities)...),
		),
		)
	}
	if container.Process.User != nil {
		opts = append(opts, oci.WithUIDGID(container.Process.User.Uid, container.Process.User.Gid))
	}
	if container.Readonly {
		opts = append(opts, oci.WithRootFSReadonly())
	}
	// make sure this opt is run after the user has been set
	opts = append(opts, withProcessCaps(container.Security.Capabilities))
	return oci.Compose(opts...)
}

func withProcessCaps(capabilities []string) oci.SpecOpts {
	return func(ctx context.Context, _ oci.Client, c *containers.Container, s *oci.Spec) error {
		if len(capabilities) == 0 {
			return nil
		}
		set := make(map[string]struct{})
		for _, s := range s.Process.Capabilities.Bounding {
			set[s] = struct{}{}
		}
		for _, cc := range capabilities {
			set[cc] = struct{}{}
		}
		ss := stringSet(set)
		s.Process.Capabilities.Bounding = ss
		s.Process.Capabilities.Effective = ss
		s.Process.Capabilities.Permitted = ss
		s.Process.Capabilities.Inheritable = ss
		if s.Process.User.UID != 0 {
			s.Process.Capabilities.Ambient = ss
		}
		return nil
	}
}

func stringSet(set map[string]struct{}) (o []string) {
	for k := range set {
		o = append(o, k)
	}
	return o
}

func ints(i []int64) (o []int) {
	for _, ii := range i {
		o = append(o, int(ii))
	}
	return o
}

func toStrings(ss []string) map[string]string {
	m := make(map[string]string, len(ss))
	for _, s := range ss {
		parts := strings.SplitN(s, "=", 2)
		m[parts[0]] = parts[1]
	}
	return m
}

func toGpuCaps(ss []string) (o []nvidia.Capability) {
	for _, s := range ss {
		o = append(o, nvidia.Capability(s))
	}
	return o
}

func withResources(r *v1.Resources) oci.SpecOpts {
	return func(ctx context.Context, _ oci.Client, c *containers.Container, s *oci.Spec) error {
		if r.Memory > 0 {
			limit := r.Memory * 1024 * 1024
			s.Linux.Resources.Memory = &specs.LinuxMemory{
				Limit: &limit,
			}
		}
		if r.Cpus > 0 {
			period := uint64(100000)
			quota := int64(r.Cpus * 100000.0)
			s.Linux.Resources.CPU = &specs.LinuxCPU{
				Quota:  &quota,
				Period: &period,
			}
		}
		if r.Score != 0 {
			score := int(r.Score)
			s.Process.OOMScoreAdj = &score
		}
		if r.NoFile > 0 {
			s.Process.Rlimits = []specs.POSIXRlimit{
				{
					Type: "RLIMIT_NOFILE",
					Hard: r.NoFile,
					Soft: r.NoFile,
				},
			}
		}
		return nil
	}
}

func withMounts(mounts []*v1.Mount) oci.SpecOpts {
	return func(ctx context.Context, _ oci.Client, c *containers.Container, s *oci.Spec) error {
		for _, cm := range mounts {
			var (
				tpe    = cm.Type
				source = cm.Source
			)
			switch cm.Type {
			case "bind":
				// create source if it does not exist
				if err := createHostDir(cm.Source, int(s.Process.User.UID), int(s.Process.User.GID)); err != nil {
					return err
				}
			case "iscsi":
				tpe = "ext4"
				// discover
				portal, iqn, err := parseISCSI(source)
				if err != nil {
					return err
				}
				if err := iscsi.Discover(ctx, portal); err != nil {
					return errors.Wrapf(err, "discover targets %s", portal)
				}
				target, err := iscsi.Login(ctx, portal, iqn)
				if err != nil {
					return errors.Wrapf(err, "unable to login %s -> %s", portal, iqn)
				}
				source = target.Lun(0)
				if err := target.Ready(1 * time.Second); err != nil {
					return errors.Wrap(err, "target not ready")
				}
			}
			s.Mounts = append(s.Mounts, specs.Mount{
				Type:        tpe,
				Source:      source,
				Destination: cm.Destination,
				Options:     cm.Options,
			})
		}
		return nil
	}
}

func parseISCSI(s string) (string, string, error) {
	parts := strings.SplitN(s, "|", 2)
	if len(parts) != 2 {
		return "", "", errors.Errorf("invalid iscsi source format %s", s)
	}
	return parts[0], parts[1], nil
}

func createHostDir(path string, uid, gid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0755); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	if err := os.Chown(path, uid, gid); err != nil {
		return err
	}
	return nil
}

func withConfigs(paths Paths, files []*v1.ConfigFile) oci.SpecOpts {
	return func(ctx context.Context, _ oci.Client, c *containers.Container, s *oci.Spec) error {
		for _, f := range files {
			s.Mounts = append(s.Mounts, specs.Mount{
				Type:        "bind",
				Source:      paths.ConfigPath(f.ID),
				Destination: f.Path,
				Options: []string{
					"ro",
					"bind",
				},
			})
		}
		return nil
	}
}

func WriteHostsFiles(root, id string) (string, string, error) {
	if err := os.MkdirAll(root, 0711); err != nil {
		return "", "", err
	}
	path := filepath.Join(root, "hosts")
	f, err := os.Create(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	if err := f.Chmod(0666); err != nil {
		return "", "", err
	}
	if _, err := f.WriteString("127.0.0.1       localhost\n"); err != nil {
		return "", "", err
	}
	if _, err := f.WriteString(fmt.Sprintf("127.0.0.1       %s\n", id)); err != nil {
		return "", "", err
	}
	if _, err := f.WriteString("::1     localhost ip6-localhost ip6-loopback\n"); err != nil {
		return "", "", err
	}
	hpath := filepath.Join(root, "hostname")
	hf, err := os.Create(hpath)
	if err != nil {
		return "", "", err
	}
	if _, err := hf.WriteString(id); err != nil {
		return "", "", err
	}
	return path, hpath, nil
}

func WithContainerHostsFile(root string) oci.SpecOpts {
	return func(ctx context.Context, _ oci.Client, c *containers.Container, s *oci.Spec) error {
		hosts, hostname, err := WriteHostsFiles(root, c.ID)
		if err != nil {
			return err
		}
		s.Mounts = append(s.Mounts, specs.Mount{
			Destination: "/etc/hosts",
			Type:        "bind",
			Source:      hosts,
			Options:     []string{"rbind", "ro"},
		})
		s.Mounts = append(s.Mounts, specs.Mount{
			Destination: "/etc/hostname",
			Type:        "bind",
			Source:      hostname,
			Options:     []string{"rbind", "ro"},
		})
		return nil
	}
}

func withOrbitResolvconf(root string) oci.SpecOpts {
	return func(ctx context.Context, _ oci.Client, c *containers.Container, s *oci.Spec) error {
		s.Mounts = append(s.Mounts, specs.Mount{
			Destination: "/etc/resolv.conf",
			Type:        "bind",
			Source:      filepath.Join(root, "resolv.conf"),
			Options:     []string{"rbind", "ro"},
		})
		return nil
	}
}

func GetConfig(ctx context.Context, container containerd.Container) (*v1.Container, error) {
	info, err := container.Info(ctx)
	if err != nil {
		return nil, err
	}
	return GetConfigFromInfo(ctx, info)
}

func GetConfigFromInfo(ctx context.Context, info containers.Container) (*v1.Container, error) {
	d := info.Extensions[CurrentConfig]
	return UnmarshalConfig(&d)
}

var ErrOldConfigFormat = errors.New("old config format on container")

func UnmarshalConfig(any *types.Any) (*v1.Container, error) {
	v, err := typeurl.UnmarshalAny(any)
	if err != nil {
		return nil, err
	}
	c, ok := v.(*v1.Container)
	if !ok {
		return nil, ErrOldConfigFormat
	}
	return c, nil
}

// WithIP sets the ip on the container
func WithIP(ip string) containerd.UpdateContainerOpts {
	return func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
		if c.Labels == nil {
			c.Labels = make(map[string]string)
		}
		c.Labels[IPLabel] = ip
		return nil
	}
}

func WithRestore(m *is.Descriptor) containerd.NewContainerOpts {
	return func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
		if c.Extensions == nil {
			c.Extensions = make(map[string]types.Any)
		}
		v := &api.Descriptor{
			MediaType: m.MediaType,
			Size_:     m.Size,
			Digest:    m.Digest,
		}
		any, err := typeurl.MarshalAny(v)
		if err != nil {
			return err
		}
		c.Extensions[RestoreCheckpointLabel] = *any
		return nil
	}
}

func WithoutRestore(ctx context.Context, client *containerd.Client, c *containers.Container) error {
	if c.Extensions == nil {
		c.Extensions = make(map[string]types.Any)
	}
	delete(c.Extensions, RestoreCheckpointLabel)
	return nil
}

func WithTaskRestore(desc *api.Descriptor) containerd.NewTaskOpts {
	return func(ctx context.Context, client *containerd.Client, ti *containerd.TaskInfo) error {
		ti.Checkpoint = desc
		return nil
	}
}

func WithISCSILogout(ctx context.Context, client *containerd.Client, c containers.Container) error {
	config, err := GetConfigFromInfo(ctx, c)
	if err != nil {
		return err
	}
	for _, m := range config.Mounts {
		if m.Type == "iscsi" {
			portal, iqn, err := parseISCSI(m.Source)
			if err != nil {
				return err
			}
			target := iscsi.LoadTarget(portal, iqn)
			if err := target.Logout(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func GetRestoreDesc(ctx context.Context, c containerd.Container) (*api.Descriptor, error) {
	ex, err := c.Extensions(ctx)
	if err != nil {
		return nil, err
	}
	any, ok := ex[RestoreCheckpointLabel]
	if !ok {
		return nil, nil
	}
	v, err := typeurl.UnmarshalAny(&any)
	if err != nil {
		return nil, err
	}
	return v.(*api.Descriptor), nil
}
