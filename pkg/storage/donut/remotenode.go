package donut

import (
	"errors"
)

type remoteNode struct {
	hostname string
	url      string
	disks    []Disk
}

// NewRemoteNode - instantiates a new remote node
func NewRemoteNode(url string) (Node, error) {
	return nil, errors.New("Not Implemented")
}

func (n remoteNode) ListDisks() ([]Disk, error) {
	return nil, errors.New("Not Implemented")
}

func (n remoteNode) AttachDisk(disk Disk) error {
	return errors.New("Not Implemented")
}

func (n remoteNode) DetachDisk(disk Disk) error {
	return errors.New("Not Implemented")
}

func (n remoteNode) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (n remoteNode) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
