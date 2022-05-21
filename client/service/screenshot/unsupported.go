//go:build !linux && !windows && !darwin

package screenshot

func GetScreenshot(bridge string) error {
	return utils.ErrUnsupported
}
