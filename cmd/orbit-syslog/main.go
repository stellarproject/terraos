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
	"bufio"
	"context"
	"fmt"
	"io"
	"log/syslog"
	"sync"

	"github.com/containerd/containerd/runtime/v2/logging"
)

func main() {
	logging.Run(log)
}

func log(ctx context.Context, config *logging.Config, ready func() error) error {
	w, err := syslog.New(syslog.LOG_DAEMON, fmt.Sprintf("%s:%s", config.Namespace, config.ID))
	if err != nil {
		return err
	}
	defer w.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	// forward both stdout and stderr to the journal
	go copy(&wg, config.Stdout, syslog.LOG_INFO, w)
	go copy(&wg, config.Stderr, syslog.LOG_ERR, w)
	// signal that we are ready and setup for the container to be started
	if err := ready(); err != nil {
		return err
	}
	wg.Wait()
	return nil
}

func copy(wg *sync.WaitGroup, r io.Reader, pri syslog.Priority, w *syslog.Writer) {
	defer wg.Done()

	s := bufio.NewScanner(r)
	for s.Scan() {
		if s.Err() != nil {
			return
		}
		var err error
		switch pri {
		case syslog.LOG_INFO:
			err = w.Info(s.Text())
		case syslog.LOG_ERR:
			err = w.Err(s.Text())
		}
		if err != nil {
			return
		}
	}
}
