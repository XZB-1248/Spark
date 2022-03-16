// +build windows

package file

import "github.com/shirou/gopsutil/v3/disk"

// ListFiles will only be called when path is root and
// current system is Windows.
// It will return mount points of all volumes.
func ListFiles(path string) ([]file, error) {
	result := make([]file, 0)
	if len(path) == 0 || path == `\` || path == `/` {
		partitions, err := disk.Partitions(true)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(partitions); i++ {
			result = append(result, file{Name: partitions[i].Mountpoint, Type: 2})
		}
		return result, nil
	}
	return listFiles(path)
}
