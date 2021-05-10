// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"

	"github.com/minio/madmin-go"
)

type hri struct {
	*madmin.HealResultItem
}

func newHRI(i *madmin.HealResultItem) *hri {
	return &hri{i}
}

// getObjectHCCChange - returns before and after color change for
// objects
func (h hri) getObjectHCCChange() (b, a col, err error) {
	parityShards := h.ParityBlocks
	dataShards := h.DataBlocks

	onlineBefore, onlineAfter := h.GetOnlineCounts()
	surplusShardsBeforeHeal := onlineBefore - dataShards
	surplusShardsAfterHeal := onlineAfter - dataShards

	b, err = getHColCode(surplusShardsBeforeHeal, parityShards)
	if err != nil {
		return
	}
	a, err = getHColCode(surplusShardsAfterHeal, parityShards)
	return

}

// getReplicatedFileHCCChange - fetches health color code for metadata
// files that are replicated.
func (h hri) getReplicatedFileHCCChange() (b, a col, err error) {
	getColCode := func(numAvail int) (c col, err error) {
		// calculate color code for replicated object similar
		// to erasure coded objects
		quorum := h.DiskCount/h.SetCount/2 + 1
		surplus := numAvail/h.SetCount - quorum
		parity := h.DiskCount/h.SetCount - quorum
		c, err = getHColCode(surplus, parity)
		return
	}

	onlineBefore, onlineAfter := h.GetOnlineCounts()
	b, err = getColCode(onlineBefore)
	if err != nil {
		return
	}
	a, err = getColCode(onlineAfter)
	return
}

func (h hri) makeHealEntityString() string {
	switch h.Type {
	case madmin.HealItemObject:
		return h.Bucket + "/" + h.Object
	case madmin.HealItemBucket:
		return h.Bucket
	case madmin.HealItemMetadata:
		return "[disk-format]"
	case madmin.HealItemBucketMetadata:
		return fmt.Sprintf("[bucket-metadata]%s/%s", h.Bucket, h.Object)
	}
	return "** unexpected **"
}

func (h hri) getHRTypeAndName() (typ, name string) {
	name = fmt.Sprintf("%s/%s", h.Bucket, h.Object)
	switch h.Type {
	case madmin.HealItemMetadata:
		typ = "system"
		name = h.Detail
	case madmin.HealItemBucketMetadata:
		typ = "system"
		name = "bucket-metadata:" + name
	case madmin.HealItemBucket:
		typ = "bucket"
	case madmin.HealItemObject:
		typ = "object"
	default:
		typ = fmt.Sprintf("!! Unknown heal result record %#v !!", h)
		name = typ
	}
	return
}

func (h hri) getHealResultStr() string {
	typ, name := h.getHRTypeAndName()

	switch h.Type {
	case madmin.HealItemMetadata, madmin.HealItemBucketMetadata:
		return typ + ":" + name
	default:
		return name
	}
}
