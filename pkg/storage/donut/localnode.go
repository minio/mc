package donut

import (
	"errors"
)

type localNode struct {
	hostname string
	disks    map[string]Disk
}

// NewLocalNode - instantiates a new local node
func NewLocalNode(hostname string) (Node, error) {
	if hostname == "" {
		return nil, errors.New("invalid argument")
	}
	disks := make(map[string]Disk)
	n := localNode{
		hostname: hostname,
		disks:    disks,
	}
	return n, nil
}

func (n localNode) GetNodeName() string {
	return n.hostname
}

func (n localNode) ListDisks() (map[string]Disk, error) {
	return n.disks, nil
}

func (n localNode) AttachDisk(disk Disk) error {
	if disk == nil {
		return errors.New("Invalid argument")
	}
	n.disks[disk.GetDiskName()] = disk
	return nil
}

func (n localNode) DetachDisk(disk Disk) error {
	delete(n.disks, disk.GetDiskName())
	return nil
}

func (n localNode) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (n localNode) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
