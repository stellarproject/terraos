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

package image

import (
	"context"
	"io"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cmd/ctr/commands/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/urfave/cli"
)

func Get(ctx context.Context, client *containerd.Client, ref string, clix *cli.Context, out io.Writer, unpack bool) (containerd.Image, error) {
	if unpack {
		fc, err := content.NewFetchConfig(ctx, clix)
		if err != nil {
			return nil, err
		}
		fc.ProgressOutput = out
		if _, err := content.Fetch(ctx, client, ref, fc); err != nil {
			return nil, err
		}
		image, err := client.GetImage(ctx, ref)
		if err != nil {
			return nil, err
		}
		if err := image.Unpack(ctx, containerd.DefaultSnapshotter); err != nil {
			return nil, err
		}
		return image, nil
	}
	image, err := client.GetImage(ctx, ref)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, err
		}
		fc, err := content.NewFetchConfig(ctx, clix)
		if err != nil {
			return nil, err
		}
		fc.ProgressOutput = out
		if _, err := content.Fetch(ctx, client, ref, fc); err != nil {
			return nil, err
		}
		if image, err = client.GetImage(ctx, ref); err != nil {
			return nil, err
		}
	}
	return image, nil
}
