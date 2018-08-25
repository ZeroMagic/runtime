// Copyright (c) 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package kata

import (
	"context"
	"github.com/containerd/containerd/mount"
	vc "github.com/kata-containers/runtime/virtcontainers"
	"github.com/sirupsen/logrus"
	"path"
)

func deleteContainer(ctx context.Context, s *service, c *container) error {

	status, err := vci.StatusContainer(ctx, s.sandbox.ID(), c.id)
	if err != nil {
		return err
	}
	if status.State.State != vc.StateStopped {
		_, err = vci.StopContainer(ctx, s.sandbox.ID(), c.id)
		if err != nil {
			return err
		}
	}

	_, err = vci.DeleteContainer(ctx, s.sandbox.ID(), c.id)
	if err != nil {
		return err
	}

	rootfs := path.Join(c.bundle, "rootfs")
	if err := mount.UnmountAll(rootfs, 0); err != nil {
		logrus.WithError(err).Warn("failed to cleanup rootfs mount")
	}

	delete(s.processes, c.pid)
	delete(s.containers, c.id)

	return nil
}
