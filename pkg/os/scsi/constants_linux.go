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

// From 2.6.x kernel onwards, no need to support procfs
var (
	SysfsScsiDevices      = "/sys/bus/scsi/devices/"
	SysfsBlock            = "/sys/block/"
	SysfsClassBlock       = "/sys/class/block/"
	SysfsClassScsiDevices = "/sys/class/scsi_device/"
	Udev                  = "/dev/"
	DevDiskByIDDir        = "/dev/disk/by-id"
)

// ScsiDeviceTypes - non exhaustive list of all the device types in linux
var ScsiDeviceTypes = []string{
	"disk",
	"tape",
	"printer",
	"process",
	"worm",
	"cd/dvd",
	"scanner",
	"optical",
	"mediumx",
	"comms",
	"(0xa)",
	"(0xb)",
	"storage",
	"enclosu",
	"sim disk",
	"optical rd",
	"bridge",
	"osd",
	"adi",
	"sec man",
	"zbc",
	"(0x15)",
	"(0x16)",
	"(0x17)",
	"(0x18)",
	"(0x19)",
	"(0x1a)",
	"(0x1b)",
	"(0x1c)",
	"(0x1e)",
	"wlun",
	"no dev",
}
