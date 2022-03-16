// +build !windows

package file

func ListFiles(path string) ([]file, error) {
	if len(path) == 0 {
		path = `/`
	}
	return listFiles(path)
}
