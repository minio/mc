package donut

import (
	"errors"
	"fmt"
	"path"
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

// attachDonutNode - wrapper function to instantiate a new node for associated donut
// based on the configuration
func (d donut) attachDonutNode(hostname string, disks []string) error {
	node, err := NewNode(hostname)
	if err != nil {
		return err
	}
	for i, disk := range disks {
		// Order is necessary for maps, keep order number separately
		newDisk, err := NewDisk(disk, i)
		if err != nil {
			return err
		}
		if err := newDisk.MakeDir(d.name); err != nil {
			return err
		}
		if err := node.AttachDisk(newDisk); err != nil {
			return err
		}
	}
	if err := d.AttachNode(node); err != nil {
		return err
	}
	return nil
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
		err := d.attachDonutNode(k, v)
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (d donut) MakeBucket(bucketName string) error {
	if bucketName == "" || strings.TrimSpace(bucketName) == "" {
		return errors.New("invalid argument")
	}
	if _, ok := d.buckets[bucketName]; ok {
		return errors.New("bucket exists")
	}
	bucket, err := NewBucket(bucketName, d.name, d.nodes)
	if err != nil {
		return err
	}
	nodeNumber := 0
	d.buckets[bucketName] = bucket
	for _, node := range d.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return err
		}
		for _, disk := range disks {
			bucketSlice := fmt.Sprintf("%s$%d$%d", bucketName, nodeNumber, disk.GetOrder())
			err := disk.MakeDir(path.Join(d.name, bucketSlice))
			if err != nil {
				return err
			}
		}
		nodeNumber = nodeNumber + 1
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
					return nil, errors.New("corrupted backend")
				}
				// we dont need this NewBucket once we cache these
				bucket, err := NewBucket(splitDir[0], d.name, d.nodes)
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

func (d donut) Info() (nodeDiskMap map[string][]string, err error) {
	nodeDiskMap = make(map[string][]string)
	for nodeName, node := range d.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return nil, err
		}
		diskList := make([]string, len(disks))
		for diskName, disk := range disks {
			diskList[disk.GetOrder()] = diskName
		}
		nodeDiskMap[nodeName] = diskList
	}
	return nodeDiskMap, nil
}

func (d donut) AttachNode(node Node) error {
	if node == nil {
		return errors.New("invalid argument")
	}
	d.nodes[node.GetNodeName()] = node
	return nil
}
func (d donut) DetachNode(node Node) error {
	delete(d.nodes, node.GetNodeName())
	return nil
}

func (d donut) SaveConfig() ([]byte, error) {
	return nil, errors.New("Not Implemented")
}

func (d donut) LoadConfig([]byte) error {
	return errors.New("Not Implemented")
}
