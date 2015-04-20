package mxj

import (
	"strings"
)

// Sets the value for the path
func (mv Map) SetValueForPath(value interface{}, path string) error {
	pathAry := strings.Split(path, ".")
	parentPathAry := pathAry[0 : len(pathAry)-1]
	parentPath := strings.Join(parentPathAry, ".")

	val, err := mv.ValueForPath(parentPath)
	if err != nil {
		return err
	}

	key := pathAry[len(pathAry)-1]
	cVal := val.(map[string]interface{})
	cVal[key] = value

	return nil
}
