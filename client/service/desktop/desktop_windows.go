package desktop

import (
	"errors"
	"github.com/kirides/go-d3d/d3d11"
	"github.com/kirides/go-d3d/outputduplication"
	"github.com/kirides/go-d3d/outputduplication/swizzle"
	winDXGI "github.com/kirides/go-d3d/win"
	winGDI "github.com/lxn/win"
	"image"
	"syscall"
	"unsafe"
)

var (
	libUser32, _               = syscall.LoadLibrary("user32.dll")
	funcGetDesktopWindow, _    = syscall.GetProcAddress(syscall.Handle(libUser32), "GetDesktopWindow")
	funcEnumDisplayMonitors, _ = syscall.GetProcAddress(syscall.Handle(libUser32), "EnumDisplayMonitors")
	funcGetMonitorInfo, _      = syscall.GetProcAddress(syscall.Handle(libUser32), "GetMonitorInfoW")
	funcEnumDisplaySettings, _ = syscall.GetProcAddress(syscall.Handle(libUser32), "EnumDisplaySettingsW")
)

type Screen struct {
	screen ScreenCapture
}
type ScreenCapture interface {
	Init(uint, image.Rectangle) error
	Capture() (*image.RGBA, error)
	Release()
}
type ScreenDXGI struct {
	rect      image.Rectangle
	device    *d3d11.ID3D11Device
	deviceCtx *d3d11.ID3D11DeviceContext
	ddup      *outputduplication.OutputDuplicator
}
type ScreenGDI struct {
	rect           image.Rectangle
	width          int
	height         int
	hwnd           winGDI.HWND
	hdc            winGDI.HDC
	memoryDevice   winGDI.HDC
	bitmap         winGDI.HBITMAP
	bitmapInfo     winGDI.BITMAPINFOHEADER
	bitmapDataSize uintptr
	hmem           winGDI.HGLOBAL
	memptr         unsafe.Pointer
}

func (s *Screen) Init(displayIndex uint, rect image.Rectangle) {
	dxgi := ScreenDXGI{}
	if dxgi.Init(displayIndex, rect) == nil {
		s.screen = &dxgi
	} else {
		gdi := ScreenGDI{}
		gdi.Init(displayIndex, rect)
		s.screen = &gdi
	}
}
func (s *Screen) Capture() (*image.RGBA, error) {
	return s.screen.Capture()
}
func (s *Screen) Release() {
	s.screen.Release()
}

func (s *ScreenDXGI) Init(displayIndex uint, rect image.Rectangle) error {
	s.rect = rect
	var err error
	if !winDXGI.IsValidDpiAwarenessContext(winDXGI.DpiAwarenessContextPerMonitorAwareV2) {
		return errors.New("no valid DPI awareness context")
	}
	_, err = winDXGI.SetThreadDpiAwarenessContext(winDXGI.DpiAwarenessContextPerMonitorAwareV2)
	if err != nil {
		return err
	}

	s.device, s.deviceCtx, err = d3d11.NewD3D11Device()
	s.ddup, err = outputduplication.NewIDXGIOutputDuplication(s.device, s.deviceCtx, displayIndex)
	if err != nil {
		s.device.Release()
		s.deviceCtx.Release()
		return err
	}
	return nil
}
func (s *ScreenDXGI) Capture() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, s.rect.Dx(), s.rect.Dy()))
	err := s.ddup.GetImage(img, 100)
	if err == outputduplication.ErrNoImageYet {
		return nil, errNoImage
	}
	return img, err
}
func (s *ScreenDXGI) Release() {
	if s.ddup != nil {
		s.ddup.Release()
		s.ddup = nil
	}
	if s.device != nil {
		s.device.Release()
		s.device = nil
	}
	if s.deviceCtx != nil {
		s.deviceCtx.Release()
		s.deviceCtx = nil
	}
}

func (s *ScreenGDI) Init(_ uint, rect image.Rectangle) error {
	s.rect = rect
	s.width = rect.Dx()
	s.height = rect.Dy()

	s.hwnd = getDesktopWindow()
	s.hdc = winGDI.GetDC(s.hwnd)
	if s.hdc == 0 {
		s.Release()
		return errors.New("GetDC failed")
	}
	s.memoryDevice = winGDI.CreateCompatibleDC(s.hdc)
	if s.memoryDevice == 0 {
		s.Release()
		return errors.New("CreateCompatibleDC failed")
	}
	s.bitmap = winGDI.CreateCompatibleBitmap(s.hdc, int32(s.width), int32(s.height))
	if s.bitmap == 0 {
		s.Release()
		return errors.New("CreateCompatibleBitmap failed")
	}

	s.bitmapInfo = winGDI.BITMAPINFOHEADER{}
	s.bitmapInfo.BiSize = uint32(unsafe.Sizeof(s.bitmapInfo))
	s.bitmapInfo.BiPlanes = 1
	s.bitmapInfo.BiBitCount = 32
	s.bitmapInfo.BiWidth = int32(s.width)
	s.bitmapInfo.BiHeight = -int32(s.height)
	s.bitmapInfo.BiCompression = winGDI.BI_RGB
	s.bitmapInfo.BiSizeImage = uint32(s.width * s.height * 4)

	s.bitmapDataSize = uintptr(((int64(s.width)*int64(s.bitmapInfo.BiBitCount) + 31) / 32) * 4 * int64(s.height))
	s.hmem = winGDI.GlobalAlloc(winGDI.GMEM_MOVEABLE, s.bitmapDataSize)
	if s.hmem == 0 {
		s.Release()
		return errors.New("GlobalAlloc failed")
	}
	s.memptr = winGDI.GlobalLock(s.hmem)
	if s.memptr == nil {
		s.Release()
		return errors.New("GlobalLock failed")
	}
	return nil
}
func (s *ScreenGDI) Capture() (*image.RGBA, error) {
	old := winGDI.SelectObject(s.memoryDevice, winGDI.HGDIOBJ(s.bitmap))
	if old == 0 {
		return nil, errors.New("SelectObject failed")
	}

	if !winGDI.BitBlt(s.memoryDevice, 0, 0, int32(s.width), int32(s.height), s.hdc, int32(s.rect.Min.X), int32(s.rect.Min.Y), winGDI.SRCCOPY) {
		return nil, errors.New("BitBlt failed")
	}

	if winGDI.GetDIBits(s.hdc, s.bitmap, 0, uint32(s.height), (*uint8)(s.memptr), (*winGDI.BITMAPINFO)(unsafe.Pointer(&s.bitmapInfo)), winGDI.DIB_RGB_COLORS) == 0 {
		return nil, errors.New("GetDIBits failed")
	}

	img := image.NewRGBA(image.Rect(0, 0, s.width, s.height))
	imageBytes := ((*[1 << 30]byte)(unsafe.Pointer(s.memptr)))[:s.bitmapDataSize:s.bitmapDataSize]
	copy(img.Pix[:s.bitmapDataSize], imageBytes)
	swizzle.BGRA(img.Pix)

	return img, nil
}
func (s *ScreenGDI) Release() {
	if s.hdc != 0 {
		winGDI.ReleaseDC(s.hwnd, s.hdc)
		s.hdc = 0
	}
	if s.memoryDevice != 0 {
		winGDI.DeleteDC(s.memoryDevice)
		s.memoryDevice = 0
	}
	if s.bitmap != 0 {
		winGDI.DeleteObject(winGDI.HGDIOBJ(s.bitmap))
		s.bitmap = 0
	}
	if s.hmem != 0 {
		winGDI.GlobalUnlock(s.hmem)
		winGDI.GlobalFree(s.hmem)
		s.hmem = 0
	}
}
func getDesktopWindow() winGDI.HWND {
	ret, _, _ := syscall.SyscallN(funcGetDesktopWindow)
	return winGDI.HWND(ret)
}
