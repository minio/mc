package donut

import (
	"errors"
)

type localNode struct {
	hostname string
	disks    map[string]Disk
}

// NewLocalNode - instantiates a new local node
func NewLocalNode() (Node, error) {
	return nil, errors.New("Not Implemented")
}

func (n localNode) GetNodeName() string {
	return n.hostname
}

func (n localNode) ListDisks() (map[string]Disk, error) {
	return nil, errors.New("Not Implemented")
}

func (n localNode) AttachDisk(disk Disk) error {
	if disk == nil {
		return errors.New("Invalid argument")
	}
	n.disks[disk.GetDiskName()] = disk
	return nil
}

func (n localNode) DetachDisk(disk Disk) error {
	return errors.New("Not Implemented")
}

func (n localNode) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (n localNode) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
