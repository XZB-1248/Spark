//go:build !windows
// +build !windows

package desktop

import (
	"github.com/kbinani/screenshot"
	"image"
)

type screen struct {
	displayIndex int
}

func (s *screen) init(displayIndex int) {
	s.displayIndex = displayIndex
}

func (s *screen) capture(img *image.RGBA, _ image.Rectangle) error {
	image, err := screenshot.CaptureDisplay(displayIndex)
	if err == nil {
		*img = *image
	}
	return err
}

func (s *screen) release() {
}
