package donut

import (
	"errors"
	"fmt"
	"strings"
)

type donut struct {
	name    string
	buckets map[string]Bucket
	nodes   map[string]Node
}

// NewDonut - instantiate a new donut
func NewDonut(donutName string) (Donut, error) {
	nodes := make(map[string]Node)
	buckets := make(map[string]Bucket)
	d := donut{
		name:    donutName,
		nodes:   nodes,
		buckets: buckets,
	}
	return d, nil
}

func (d donut) MakeBucket(bucketName string) error {
	bucket, err := NewBucket(bucketName)
	if err != nil {
		return err
	}
	i := 0
	d.buckets[bucketName] = bucket
	for _, node := range d.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return err
		}
		for _, disk := range disks {
			bucketSlice := fmt.Sprintf("%s/%s$%d$", d.name, bucketName, i)
			err := disk.MakeDir(bucketSlice)
			if err != nil {
				return err
			}
		}
		i = i + 1
	}
	return nil
}

func (d donut) ListBuckets() (map[string]Bucket, error) {
	for _, node := range d.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return nil, err
		}
		for _, disk := range disks {
			dirs, err := disk.ListDir(d.name)
			if err != nil {
				return nil, err
			}
			for _, dir := range dirs {
				splitDir := strings.Split(dir.Name(), "$")
				if len(splitDir) < 3 {
					return nil, errors.New("Corrupted backend")
				}
				bucket, err := NewBucket(splitDir[0])
				if err != nil {
					return nil, err
				}
				d.buckets[bucket.GetBucketName()] = bucket
			}
		}
	}
	return d.buckets, nil
}

func (d donut) Heal() error {
	return errors.New("Not Implemented")
}

func (d donut) Rebalance() error {
	return errors.New("Not Implemented")
}

func (d donut) Info() error {
	return errors.New("Not Implemented")
}

func (d donut) AttachNode(node Node) error {
	if node == nil {
		return errors.New("invalid argument")
	}
	d.nodes[node.GetNodeName()] = node
	return nil
}
func (d donut) DetachNode(node Node) error {
	return errors.New("Not Implemented")
}

func (d donut) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (d donut) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
