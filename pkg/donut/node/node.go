package node

import (
	"errors"

	"github.com/minio-io/mc/pkg/os/scsi"
)

type Node struct {
	*node
}

type node struct {
	devices *scsi.Devices
	bucket  string
}

func New(bucket string) *Node {
	n := &Node{&node{bucket: bucket}}
	return n
}

func (n *Node) GetDisks() error {
	if n == nil {
		return errors.New("invalid argument")
	}
	n.devices = &scsi.Devices{}
	if err := n.devices.Get(); err != nil {
		return err
	}
	return nil
}

func (n *Node) CreateBucket() error {
}
