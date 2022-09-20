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
	var err error
	img, err = screenshot.CaptureDisplay(displayIndex)
	return err
}

func (s *screen) release() {
}
