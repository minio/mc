package donut

import (
	"errors"
)

type donut struct {
	buckets map[string]Bucket
	nodes   map[string]Node
}

// NewDonut - instantiate a new donut
func NewDonut(donutName string, nodes map[string]Node) (Donut, error) {
	return nil, errors.New("Not Implemented")
}

func (d donut) MakeBucket(bucket string) error {
	return errors.New("Not Implemented")
}

func (d donut) ListBuckets() (map[string]Bucket, error) {
	return nil, errors.New("Not Implemented")
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
	return errors.New("Not Implemented")
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
