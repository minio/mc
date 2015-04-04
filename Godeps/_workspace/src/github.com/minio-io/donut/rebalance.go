package donut

import "fmt"

func (d donut) Rebalance() error {
	for _, node := range d.nodes {
		disks, err := node.ListDisks()
		if err != nil {
			return err
		}
		fmt.Println(len(disks))
	}
	return nil
}
