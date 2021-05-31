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
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// HwServersV1 - hardware health Info
type HwServersV1 struct {
	Servers []HwServerV1 `json:"servers,omitempty"`
}

// HwServerV1 - server health Info
type HwServerV1 struct {
	Addr    string      `json:"addr"`
	CPUs    []HwCPUV1   `json:"cpus,omitempty"`
	Drives  []HwDriveV1 `json:"drives,omitempty"`
	MemInfo HwMemV1     `json:"meminfo,omitempty"`
	Perf    HwPerfV1    `json:"perf,omitempty"`
}

// HwCPUV1 - CPU Info
type HwCPUV1 struct {
	CPUStat   []cpu.InfoStat  `json:"cpu,omitempty"`
	TimesStat []cpu.TimesStat `json:"time,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// HwDriveV1 - Drive info
type HwDriveV1 struct {
	Counters   map[string]disk.IOCountersStat `json:"counters,omitempty"`
	Partitions []madmin.PartitionStat         `json:"partitions,omitempty"`
	Usage      []*disk.UsageStat              `json:"usage,omitempty"`
	Error      string                         `json:"error,omitempty"`
}

// HwMemV1 - Includes host virtual and swap mem information
type HwMemV1 struct {
	SwapMem    *mem.SwapMemoryStat    `json:"swap,omitempty"`
	VirtualMem *mem.VirtualMemoryStat `json:"virtualmem,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// HwPerfV1 - hardware performance
type HwPerfV1 struct {
	Net   HwNetPerfV1   `json:"net,omitempty"`
	Drive HwDrivePerfV1 `json:"drives,omitempty"`
}

// HwNetPerfV1 - Network performance info
type HwNetPerfV1 struct {
	Serial   []madmin.NetPerfInfoV0 `json:"serial,omitempty"`
	Parallel []madmin.NetPerfInfoV0 `json:"parallel,omitempty"`
}

// HwDrivePerfV1 - Network performance info
type HwDrivePerfV1 struct {
	Serial   []madmin.DrivePerfInfoV0 `json:"serial,omitempty"`
	Parallel []madmin.DrivePerfInfoV0 `json:"parallel,omitempty"`
	Error    string                   `json:"error,omitempty"`
}

// SwInfoV1 - software health Info
type SwInfoV1 struct {
	Minio  MinioHealthInfoV1     `json:"minio,omitempty"`
	OsInfo []madmin.ServerOsInfo `json:"osinfos,omitempty"`
}

// MinioHealthInfoV1 - Health info of the MinIO cluster
type MinioHealthInfoV1 struct {
	Info     madmin.InfoMessage      `json:"info,omitempty"`
	Config   interface{}             `json:"config,omitempty"`
	ProcInfo []madmin.ServerProcInfo `json:"procinfos,omitempty"`
	Error    string                  `json:"error,omitempty"`
}

// ClusterHealthV1 - main struct of the health report
type ClusterHealthV1 struct {
	TimeStamp time.Time   `json:"timestamp,omitempty"`
	Status    string      `json:"status"`
	Error     string      `json:"error,omitempty"`
	Hardware  HwServersV1 `json:"hardware,omitempty"`
	Software  SwInfoV1    `json:"software,omitempty"`
}

func (ch ClusterHealthV1) String() string {
	return ch.JSON()
}

// JSON jsonifies service status message.
func (ch ClusterHealthV1) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(ch, " ", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// GetStatus - return status of the health info
func (ch ClusterHealthV1) GetStatus() string {
	return ch.Status
}

// GetError - return error from the health info
func (ch ClusterHealthV1) GetError() string {
	return ch.Error
}

// GetTimestamp - return timestamp from the health info
func (ch ClusterHealthV1) GetTimestamp() time.Time {
	return ch.TimeStamp
}

// MapHealthInfoToV1 - maps the health info returned by minio server to V1 format
func MapHealthInfoToV1(healthInfo madmin.HealthInfoV0, err error) HealthReportInfo {
	ch := ClusterHealthV1{}
	ch.TimeStamp = healthInfo.TimeStamp
	if err != nil {
		ch.Status = "Error"
		ch.Error = err.Error()
		return ch
	}

	ch.Status = "Success"

	serverAddrs := set.NewStringSet()

	var serverCPUs map[string][]HwCPUV1
	var serverDrives map[string][]HwDriveV1
	var serverMems map[string]HwMemV1
	var serverNetPerfSerial, serverNetPerfParallel map[string][]madmin.NetPerfInfoV0
	var serverDrivePerf map[string]HwDrivePerfV1

	mapCPUs := func() { serverCPUs = mapServerCPUs(healthInfo) }
	mapDrives := func() { serverDrives = mapServerDrives(healthInfo) }
	mapMems := func() { serverMems = mapServerMems(healthInfo) }
	mapNetPerf := func() { serverNetPerfSerial, serverNetPerfParallel = mapServerNetPerf(healthInfo) }
	mapDrivePerf := func() { serverDrivePerf = mapServerDrivePerf(healthInfo) }

	parallelize(mapCPUs, mapDrives, mapMems, mapNetPerf, mapDrivePerf)

	addKeysToSet(reflect.ValueOf(serverCPUs).MapKeys(), &serverAddrs)
	addKeysToSet(reflect.ValueOf(serverDrives).MapKeys(), &serverAddrs)
	addKeysToSet(reflect.ValueOf(serverMems).MapKeys(), &serverAddrs)
	addKeysToSet(reflect.ValueOf(serverNetPerfSerial).MapKeys(), &serverAddrs)
	if len(healthInfo.Perf.NetParallel.Addr) > 0 {
		serverAddrs.Add(healthInfo.Perf.NetParallel.Addr)
	}
	addKeysToSet(reflect.ValueOf(serverDrivePerf).MapKeys(), &serverAddrs)

	// Merge hardware info
	hw := HwServersV1{Servers: []HwServerV1{}}
	for addr := range serverAddrs {
		perf := HwPerfV1{
			Net: HwNetPerfV1{
				Serial:   serverNetPerfSerial[addr],
				Parallel: serverNetPerfParallel[addr],
			},
			Drive: serverDrivePerf[addr],
		}
		hw.Servers = append(hw.Servers, HwServerV1{
			Addr:    addr,
			CPUs:    serverCPUs[addr],
			Drives:  serverDrives[addr],
			MemInfo: serverMems[addr],
			Perf:    perf,
		})
	}

	ch.Hardware = hw

	ch.Software = SwInfoV1{
		Minio: MinioHealthInfoV1{
			Info:     healthInfo.Minio.Info,
			Config:   healthInfo.Minio.Config,
			Error:    healthInfo.Minio.Error,
			ProcInfo: healthInfo.Sys.ProcInfo,
		},
		OsInfo: healthInfo.Sys.OsInfo,
	}
	return ch
}

func parallelize(functions ...func()) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(functions))

	defer waitGroup.Wait()

	for _, function := range functions {
		go func(fn func()) {
			defer waitGroup.Done()
			fn()
		}(function)
	}
}

func addKeysToSet(input []reflect.Value, output *set.StringSet) {
	for _, key := range input {
		output.Add(key.String())
	}
}

func mapServerCPUs(healthInfo madmin.HealthInfoV0) map[string][]HwCPUV1 {
	serverCPUs := map[string][]HwCPUV1{}
	for _, ci := range healthInfo.Sys.CPUInfo {
		cpus, ok := serverCPUs[ci.Addr]
		if !ok {
			cpus = []HwCPUV1{}
		}
		cpus = append(cpus, HwCPUV1{
			CPUStat:   ci.CPUStat,
			TimesStat: ci.TimeStat,
			Error:     ci.Error,
		})
		serverCPUs[ci.Addr] = cpus
	}
	return serverCPUs
}

func mapServerDrives(healthInfo madmin.HealthInfoV0) map[string][]HwDriveV1 {
	serverDrives := map[string][]HwDriveV1{}
	for _, di := range healthInfo.Sys.DiskHwInfo {
		drives, ok := serverDrives[di.Addr]
		if !ok {
			drives = []HwDriveV1{}
		}
		drives = append(drives, HwDriveV1{
			Counters:   di.Counters,
			Partitions: di.Partitions,
			Usage:      di.Usage,
			Error:      di.Error,
		})
		serverDrives[di.Addr] = drives
	}
	return serverDrives
}

func mapServerMems(healthInfo madmin.HealthInfoV0) map[string]HwMemV1 {
	serverMems := map[string]HwMemV1{}
	for _, mi := range healthInfo.Sys.MemInfo {
		serverMems[mi.Addr] = HwMemV1{
			SwapMem:    mi.SwapMem,
			VirtualMem: mi.VirtualMem,
			Error:      mi.Error,
		}
	}
	return serverMems
}

func mapServerNetPerf(healthInfo madmin.HealthInfoV0) (map[string][]madmin.NetPerfInfoV0, map[string][]madmin.NetPerfInfoV0) {
	snpSerial := map[string][]madmin.NetPerfInfoV0{}
	for _, serverPerf := range healthInfo.Perf.Net {
		snpSerial[serverPerf.Addr] = serverPerf.Net
	}

	snpParallel := map[string][]madmin.NetPerfInfoV0{}
	snpParallel[healthInfo.Perf.NetParallel.Addr] = healthInfo.Perf.NetParallel.Net

	return snpSerial, snpParallel
}

func mapServerDrivePerf(healthInfo madmin.HealthInfoV0) map[string]HwDrivePerfV1 {
	sdp := map[string]HwDrivePerfV1{}
	for _, drivePerf := range healthInfo.Perf.DriveInfo {
		sdp[drivePerf.Addr] = HwDrivePerfV1{
			Serial:   drivePerf.Serial,
			Parallel: drivePerf.Parallel,
			Error:    drivePerf.Error,
		}
	}
	return sdp
}
