//go:build windows
// +build windows

package desktop

import (
	"github.com/kirides/screencapture/d3d"
	"github.com/kirides/screencapture/screenshot"
	"github.com/kirides/screencapture/win"
	"image"
)

type screen struct {
	dxgi         bool
	ddup         *d3d.OutputDuplicator
	device       *d3d.ID3D11Device
	deviceCtx    *d3d.ID3D11DeviceContext
	displayIndex int
}

func (s *screen) init(displayIndex int) {
	var err error
	s.displayIndex = displayIndex
	s.dxgi = false
	return
	if win.IsValidDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2) {
		_, err = win.SetThreadDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2)
		s.dxgi = err == nil
	}
	if s.dxgi {
		s.device, s.deviceCtx, err = d3d.NewD3D11Device()
		s.ddup, err = d3d.NewIDXGIOutputDuplication(s.device, s.deviceCtx, uint(displayIndex))
		if err != nil {
			s.dxgi = false
			s.device.Release()
			s.deviceCtx.Release()
		}
	}
}

func (s *screen) capture(img *image.RGBA, bounds image.Rectangle) error {
	var err error
	if s.dxgi {
		err = s.ddup.GetImage(img, 100)
		if err != nil {
			if err == d3d.ErrNoImageYet {
				return ErrNoImage
			}
			return err
		}
		return nil
	}
	return screenshot.CaptureImg(img, 0, 0, bounds.Dx(), bounds.Dy())
}

func (s *screen) release() {
	if s.ddup != nil {
		s.ddup.Release()
	}
	if s.device != nil {
		s.device.Release()
	}
	if s.deviceCtx != nil {
		s.deviceCtx.Release()
	}
}
