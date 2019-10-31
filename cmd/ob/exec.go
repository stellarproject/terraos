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
	"errors"
	"io"
	"os"

	"github.com/containerd/console"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/cmd/ctr/commands/tasks"
	"github.com/containerd/containerd/defaults"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a process inside a container",
	Action: func(clix *cli.Context) error {
		var (
			id   = clix.Args().First()
			args = clix.Args().Tail()
			ctx  = Context()
		)
		if id == "" {
			return errors.New("id must not be empty")
		}
		client, err := containerd.New(defaults.DefaultAddress)
		if err != nil {
			return err
		}
		defer client.Close()

		container, err := client.LoadContainer(ctx, id)
		if err != nil {
			return err
		}
		spec, err := container.Spec(ctx)
		if err != nil {
			return err
		}
		task, err := container.Task(ctx, nil)
		if err != nil {
			return err
		}

		pspec := spec.Process
		pspec.Terminal = true
		pspec.Args = args

		stdinC := &stdinCloser{
			stdin: os.Stdin,
		}

		ioCreator := cio.NewCreator(
			cio.WithStreams(stdinC, os.Stdout, os.Stderr),
			cio.WithFIFODir(""),
			cio.WithTerminal,
		)

		uuid := uuid.New().String()

		process, err := task.Exec(ctx, uuid, pspec, ioCreator)
		if err != nil {
			return err
		}
		defer process.Delete(ctx)

		stdinC.closer = func() {
			process.CloseIO(ctx, containerd.WithStdinCloser)
		}

		statusC, err := process.Wait(ctx)
		if err != nil {
			return err
		}

		con := console.Current()
		defer con.Reset()
		if err := con.SetRaw(); err != nil {
			return err
		}

		if err := tasks.HandleConsoleResize(ctx, process, con); err != nil {
			logrus.WithError(err).Error("console resize")
		}
		if err := process.Start(ctx); err != nil {
			return err
		}
		status := <-statusC
		code, _, err := status.Result()
		if err != nil {
			return err
		}
		if code != 0 {
			return cli.NewExitError("", int(code))
		}
		return nil
	},
}

type stdinCloser struct {
	stdin  *os.File
	closer func()
}

func (s *stdinCloser) Read(p []byte) (int, error) {
	n, err := s.stdin.Read(p)
	if err == io.EOF {
		if s.closer != nil {
			s.closer()
		}
	}
	return n, err
}
