package donut

import (
	"errors"
)

type node struct {
	hostname string
	disks    map[string]Disk
}

// NewNode - instantiates a new node
func NewNode(hostname string) (Node, error) {
	if hostname == "" {
		return nil, errors.New("invalid argument")
	}
	disks := make(map[string]Disk)
	n := node{
		hostname: hostname,
		disks:    disks,
	}
	return n, nil
}

func (n node) GetNodeName() string {
	return n.hostname
}

func (n node) ListDisks() (map[string]Disk, error) {
	return n.disks, nil
}

func (n node) AttachDisk(disk Disk) error {
	if disk == nil {
		return errors.New("Invalid argument")
	}
	n.disks[disk.GetName()] = disk
	return nil
}

func (n node) DetachDisk(disk Disk) error {
	delete(n.disks, disk.GetName())
	return nil
}

func (n node) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (n node) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
