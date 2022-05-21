//go:build !windows
// +build !windows

package file

func ListFiles(path string) ([]File, error) {
	if len(path) == 0 {
		path = `/`
	}
	return listFiles(path)
}
