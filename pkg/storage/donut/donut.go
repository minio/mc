package donut

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Total allowed disks per node
const (
	disksPerNode = 16
)

type donut struct {
	name    string
	buckets map[string]Bucket
	nodes   map[string]Node
}

// setDonutNode - wrapper function to instantiate a new donut node based on the configuration
func setDonutNode(hostname string, disks []string) (Node, error) {
	node, err := NewNode(hostname)
	if err != nil {
		return nil, err
	}
	for i, disk := range disks {
		// Order is necessary for maps, keep order number separately
		newDisk, err := NewDisk(disk, strconv.Itoa(i))
		if err != nil {
			return nil, err
		}
		if err := node.AttachDisk(newDisk); err != nil {
			return nil, err
		}
	}
	return node, nil
}

// NewDonut - instantiate a new donut
func NewDonut(donutName string, nodeDiskMap map[string][]string) (Donut, error) {
	if donutName == "" || len(nodeDiskMap) == 0 {
		return nil, errors.New("invalid arguments")
	}

	nodes := make(map[string]Node)
	buckets := make(map[string]Bucket)
	d := donut{
		name:    donutName,
		nodes:   nodes,
		buckets: buckets,
	}

	for k, v := range nodeDiskMap {
		if len(v) > disksPerNode || len(v) == 0 {
			return nil, errors.New("invalid number of disks per node")
		}
		n, err := setDonutNode(k, v)
		if err != nil {
			return nil, err
		}
		if err := d.AttachNode(n); err != nil {
			return nil, err
		}
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
