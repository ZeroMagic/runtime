// Copyright (c) 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package kata

import (
	"context"
	"fmt"
	"github.com/containerd/containerd/api/types/task"
)

func startContainer(ctx context.Context, s *service, c *container) error {
	//start a container
	if c.cType == "" {
		err := fmt.Errorf("Bug, the container %s type is empty", c.id)
		return err
	}

	if s.sandbox == nil {
		err := fmt.Errorf("Bug, the sandbox hasn't been created for this container %s", c.id)
		return err
	}

	if c.cType.IsSandbox() {
		_, err := vci.StartSandbox(ctx, s.sandbox.ID())
		if err != nil {
			return err
		}
	} else {
		_, err := vci.StartContainer(ctx, s.sandbox.ID(), c.id)
		if err != nil {
			return err
		}
	}

	c.status = task.StatusRunning

	stdin, stdout, stderr, err := s.sandbox.IOStream(c.id, c.id)
	if err != nil {
		return err
	}

	if c.stdin != "" || c.stdout != "" || c.stderr != "" {
		tty, err := newTtyIO(ctx, c.stdin, c.stdout, c.stderr, c.terminal)
		if err != nil {
			return err
		}
		c.ttyio = tty
		go ioCopy(c.exitIOch, tty, stdin, stdout, stderr)
	} else {
		//close the io exit channel, since there is no io for this container,
		//otherwise the following wait goroutine will hang on this channel.
		close(c.exitIOch)
	}

	go wait(s, c, "")

	return nil
}

func startExec(ctx context.Context, s *service, containerID, execID string) (*exec, error) {
	//start an exec
	c, err := s.getContainer(containerID)
	if err != nil {
		return nil, err
	}

	execs, err := c.getExec(execID)
	if err != nil {
		return nil, err
	}

	_, proc, err := s.sandbox.EnterContainer(containerID, *execs.cmds)
	if err != nil {
		err := fmt.Errorf("cannot enter container %s, with err %s", containerID, err)
		return nil, err
	}
	execs.id = proc.Token
	pid := s.pid()
	execs.pid = pid
	s.processes[pid] = execID

	execs.status = task.StatusRunning
	if execs.tty.height != 0 && execs.tty.width != 0 {
		err = s.sandbox.WinsizeProcess(c.id, execs.id, execs.tty.height, execs.tty.width)
		if err != nil {
			return nil, err
		}
	}

	stdin, stdout, stderr, err := s.sandbox.IOStream(c.id, execs.id)
	if err != nil {
		return nil, err
	}
	tty, err := newTtyIO(ctx, execs.tty.stdin, execs.tty.stdout, execs.tty.stderr, execs.tty.terminal)
	if err != nil {
		return nil, err
	}
	execs.ttyio = tty

	go ioCopy(execs.exitIOch, tty, stdin, stdout, stderr)

	go wait(s, c, execID)

	return execs, nil
}
