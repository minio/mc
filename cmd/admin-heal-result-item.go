/*
 * MinIO Client (C) 2018 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"

	"github.com/minio/minio/pkg/madmin"
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
