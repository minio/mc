// Copyright (c) 2015-2022 MinIO, Inc.
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
	"errors"
	"fmt"

	"github.com/minio/madmin-go/v3"
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
		err = fmt.Errorf("%w: surplusShardsBeforeHeal: %d, parityShards: %d",
			err, surplusShardsBeforeHeal, parityShards)
		return
	}
	a, err = getHColCode(surplusShardsAfterHeal, parityShards)
	if err != nil {
		err = fmt.Errorf("%w: surplusShardsAfterHeal: %d, parityShards: %d",
			err, surplusShardsAfterHeal, parityShards)
	}
	return
}

// getBucketHCCChange - fetches health color code for bucket healing
// this does not return a Grey color since it does not have any meaning
// for a bucket healing. Return green if the bucket is found in a drive,
// yellow for missing, and red for everything else, grey for weird situations
func (h hri) getBucketHCCChange() (b, a col, err error) {
	if h.HealResultItem == nil {
		return colGrey, colGrey, errors.New("empty result")
	}

	getColCode := func(drives []madmin.HealDriveInfo) (c col) {
		var missing, unavailable int
		for i := range drives {
			switch drives[i].State {
			case madmin.DriveStateOk:
			case madmin.DriveStateMissing:
				missing++
			default:
				unavailable++
			}
		}
		if unavailable > 0 {
			return colRed
		}
		if missing > 0 {
			return colYellow
		}
		return colGreen
	}

	a, b = colGrey, colGrey

	if len(h.Before.Drives) > 0 {
		b = getColCode(h.Before.Drives)
	}
	if len(h.After.Drives) > 0 {
		a = getColCode(h.After.Drives)
	}
	return
}

// getReplicatedFileHCCChange - fetches health color code for metadata
// files that are replicated.
func (h hri) getReplicatedFileHCCChange() (b, a col, err error) {
	getColCode := func(numAvail int) (c col, err error) {
		// calculate color code for replicated object similar
		// to erasure coded objects
		var quorum, surplus, parity int
		if h.SetCount > 0 {
			quorum = h.DiskCount/h.SetCount/2 + 1
			surplus = numAvail/h.SetCount - quorum
			parity = h.DiskCount/h.SetCount - quorum
		} else {
			// in case of bucket healing, disk count is for the node
			// also explicitly set count would be set to invalid value of -1
			quorum = h.DiskCount/2 + 1
			surplus = numAvail - quorum
			parity = h.DiskCount - quorum
		}
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
