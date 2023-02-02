//go:build !windows
// +build !windows

package desktop

import (
	"github.com/kbinani/screenshot"
	"image"
)

type Screen struct {
	rect image.Rectangle
}

func (s *Screen) Init(_ uint, rect image.Rectangle) {
	s.rect = rect
}

func (s *Screen) Capture() (*image.RGBA, error) {
	return screenshot.CaptureRect(s.rect)
}

func (s *Screen) Release() {}
