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

package scsi

// NoDiskAttributesFound - no disk attributes found
type NoDiskAttributesFound struct{}

func (e NoDiskAttributesFound) Error() string {
	return "No Disk Attributes Found"
}

// NoDiskQueueAttributesFound - no disk queue attributes found
type NoDiskQueueAttributesFound struct{}

func (e NoDiskQueueAttributesFound) Error() string {
	return "No Disk Queue Attributes Found"
}

// NoDevicesFoundOnSystem - no scsi disks found on a given system
type NoDevicesFoundOnSystem struct{}

func (e NoDevicesFoundOnSystem) Error() string {
	return "No scsi devices found on the system"
}

// AddressPointsToMultipleBlockDevices - scsi address points to multiple block devices
type AddressPointsToMultipleBlockDevices struct{}

func (e AddressPointsToMultipleBlockDevices) Error() string {
	return "Scsi address points to multiple block devices"
}

// NoAttributesFound - no disk scsi attributes found
type NoAttributesFound struct{}

func (e NoAttributesFound) Error() string {
	return "No Scsi Attributes Found"
}
