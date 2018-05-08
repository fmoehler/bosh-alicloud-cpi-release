/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package action

import (
	"bosh-alicloud-cpi/alicloud"

	"github.com/cppforlife/bosh-cpi-go/apiv1"
)

type DeleteSnapshotMethod struct {
	CallContext
	disks alicloud.DiskManager
}

func NewDeleteSnapshotMethod(cc CallContext, disks alicloud.DiskManager) DeleteSnapshotMethod {
	return DeleteSnapshotMethod{cc, disks}
}

func (a DeleteSnapshotMethod) DeleteSnapshot(snapshotCID apiv1.SnapshotCID) error {
	cid := snapshotCID.AsString()

	err := a.disks.DeleteSnapshot(cid)

	if err != nil {
		return a.WrapErrorf(err, "delete snapshot %s failed", cid)
	}

	return nil
}
