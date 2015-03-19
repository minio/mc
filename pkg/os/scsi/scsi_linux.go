/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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

// !build linux,amd64

package scsi

import (
	"io/ioutil"
	"path"
)

// NOTE : supporting virtio based scsi devices
//        is out of scope for this implementation

// Devices - list of all scsi disks
type Devices struct {
	List []Disk
}

// Disk - struct which carries per disk name, scsi and disk attributes
type Disk struct {
	Name        string
	Scsiattrmap map[string][]byte
	Diskattrmap map[string][]byte
}

// getDisk - populates all the disk related attributes
func (d *Devices) getDisk(disk string) (map[string][]byte, error) {
	var diskAttrsList []string
	var diskQueueAttrs []string
	aggrAttrMap := make(map[string][]byte)

	sysfsBlockDev := path.Join(SysfsBlock, disk)
	sysfsBlockDevQueue := path.Join(sysfsBlockDev, "/queue")

	scsiFiles, err := ioutil.ReadDir(sysfsBlockDev)
	if err != nil {
		return nil, err
	}

	scsiQueueFiles, err := ioutil.ReadDir(sysfsBlockDevQueue)
	if err != nil {
		return nil, err
	}

	for _, sf := range scsiFiles {
		if sf.IsDir() {
			continue
		}
		// Skip symlinks
		if !sf.Mode().IsRegular() {
			continue
		}
		// Skip, not readable, write-only
		if sf.Mode().Perm() == 128 {
			continue
		}
		diskAttrsList = append(diskAttrsList, sf.Name())
	}

	for _, sf := range scsiQueueFiles {
		if sf.IsDir() {
			continue
		}
		// Skip symlinks
		if !sf.Mode().IsRegular() {
			continue
		}
		// Skip, not readable, write-only
		if sf.Mode().Perm() == 128 {
			continue
		}
		diskQueueAttrs = append(diskQueueAttrs, sf.Name())
	}

	if len(diskAttrsList) == 0 {
		return nil, NoDiskAttributesFound{}
	}

	if len(diskQueueAttrs) == 0 {
		return nil, NoDiskQueueAttributesFound{}
	}

	diskAttrMap := getattrs(sysfsBlockDev, diskAttrsList)
	diskQueueAttrMap := getattrs(sysfsBlockDevQueue, diskQueueAttrs)

	for k, v := range diskAttrMap {
		aggrAttrMap[k] = v
	}

	for k, v := range diskQueueAttrMap {
		aggrAttrMap[k] = v
	}

	return aggrAttrMap, nil
}

// Get - get queries local system and populates all the attributes
func (d *Devices) Get() error {
	var scsidevices []string
	var scsiAttrList []string

	sysFiles, err := ioutil.ReadDir(SysfsScsiDevices)
	if err != nil {
		return err
	}

	scsidevices = filterdevices(sysFiles)
	if len(scsidevices) == 0 {
		return NoDevicesFoundOnSystem{}
	}

	for _, scsi := range scsidevices {
		var _scsi Disk
		scsiAttrPath := path.Join(SysfsScsiDevices, scsi, "/")
		scsiAttrs, err := ioutil.ReadDir(scsiAttrPath)
		if err != nil {
			return err
		}
		scsiBlockPath := path.Join(SysfsScsiDevices, scsi, "/block")
		scsidevList, err := ioutil.ReadDir(scsiBlockPath)
		if err != nil {
			return err
		}

		if len(scsidevList) > 1 {
			return AddressPointsToMultipleBlockDevices{}
		}

		_scsi.Name = path.Join(Udev, scsidevList[0].Name())
		for _, sa := range scsiAttrs {
			// Skip directories
			if sa.IsDir() {
				continue
			}
			// Skip symlinks
			if !sa.Mode().IsRegular() {
				continue
			}
			// Skip, not readable, write-only
			if sa.Mode().Perm() == 128 {
				continue
			}
			scsiAttrList = append(scsiAttrList, sa.Name())
		}

		if len(scsiAttrList) == 0 {
			return NoAttributesFound{}
		}
		attrMap := getattrs(scsiAttrPath, scsiAttrList)
		_scsi.Scsiattrmap = attrMap
		_scsi.Diskattrmap, err = d.getDisk(scsidevList[0].Name())
		if err != nil {
			return err
		}
		d.List = append(d.List, _scsi)
	}
	return nil
}
