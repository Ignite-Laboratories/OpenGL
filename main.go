package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"image"
	"os/signal"

	"os"
	"time"
	"unsafe"

	_ "image/jpeg"

	"github.com/NeowayLabs/drm"
	"github.com/NeowayLabs/drm/mode"
)

type (
	framebuffer struct {
		id     uint32
		handle uint32
		data   []byte
		fb     *mode.FB
		size   uint64
		stride uint32
	}

	// msetData just store the pair (mode, fb) and the saved CRTC of the mode.
	msetData struct {
		mode      *mode.Modeset
		fb        framebuffer
		savedCrtc *mode.Crtc
	}

	vblankData struct {
		Type      uint32
		Sequence  uint32
		Time_sec  uint64
		Time_usec uint64
		Reserved  uint32 // This field is important for alignment
	}
)

const (
	DRM_VBLANK_RELATIVE        = 0x1
	DRM_VBLANK_EVENT           = 0x4
	DRM_VBLANK_HIGH_CRTC_SHIFT = 1
	DRM_IOCTL_WAIT_VBLANK      = 0x40406420
)

func waitVBlank(file *os.File, crtcID uint32) error {
	// Calculate the proper type value including the CRTC index
	vblankType := uint32(DRM_VBLANK_RELATIVE)
	if crtcID > 1 {
		// For CRTC indices > 1, we need to use the high CRTC mechanism
		vblankType |= (crtcID << DRM_VBLANK_HIGH_CRTC_SHIFT)
	}

	vbl := vblankData{
		Type:     vblankType,
		Sequence: 1, // Wait for next vblank
	}

	_, _, err := unix.Syscall(unix.SYS_IOCTL,
		file.Fd(),
		DRM_IOCTL_WAIT_VBLANK,
		uintptr(unsafe.Pointer(&vbl)))

	if err != 0 {
		return fmt.Errorf("vblank wait failed: %v", err)
	}

	return nil
}

func createFramebuffer(file *os.File, dev *mode.Modeset) (framebuffer, error) {
	fb, err := mode.CreateFB(file, dev.Width, dev.Height, 32)
	if err != nil {
		return framebuffer{}, fmt.Errorf("Failed to create framebuffer: %s", err.Error())
	}
	stride := fb.Pitch
	size := fb.Size
	handle := fb.Handle

	fbID, err := mode.AddFB(file, dev.Width, dev.Height, 24, 32, stride, handle)
	if err != nil {
		return framebuffer{}, fmt.Errorf("Cannot create dumb buffer: %s", err.Error())
	}

	offset, err := mode.MapDumb(file, handle)
	if err != nil {
		return framebuffer{}, err
	}

	mmap, err := unix.Mmap(int(file.Fd()), int64(offset), int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return framebuffer{}, fmt.Errorf("Failed to mmap framebuffer: %s", err.Error())
	}
	for i := uint64(0); i < size; i++ {
		mmap[i] = 0
	}
	framebuf := framebuffer{
		id:     fbID,
		handle: handle,
		data:   mmap,
		fb:     fb,
		size:   size,
		stride: stride,
	}
	return framebuf, nil
}

func renderLoop(file *os.File, msets []msetData) {
	running := true

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		running = false
	}()

	// Get the CRTC index (should be smaller than the ID)
	crtcIndex := uint32(0)
	if len(msets) > 0 {
		// You might need to calculate the actual index based on your setup
		// Usually it's a small number (0, 1, 2) rather than the large ID
		crtcIndex = msets[0].mode.Crtc % 32 // Assuming max 32 CRTCs
		fmt.Printf("Using CRTC Index: %d (from ID: %d)\n", crtcIndex, msets[0].mode.Crtc)
	}

	for running {
		// Wait for VBlank using the CRTC index
		if len(msets) > 0 {
			err := waitVBlank(file, crtcIndex)
			if err != nil {
				fmt.Fprintf(os.Stderr, "VBlank wait error: %s\n", err.Error())
			}
		}

		// Rest of your rendering code remains the same
		for _, mset := range msets {
			clearFramebuffer(mset.fb)
		}

		for j := 0; j < len(msets); j++ {
			mset := msets[j]
			t := time.Now().UnixNano() / 1000000
			for y := uint32(0); y < uint32(mset.mode.Height); y++ {
				for x := uint32(0); x < uint32(mset.mode.Width); x++ {
					off := (mset.fb.stride * y) + (x * 4)
					val := uint32(((x + uint32(t)) ^ y) & 0xFF)
					color := uint32((val << 16) | (val << 8) | val)
					*(*uint32)(unsafe.Pointer(&mset.fb.data[off])) = color
				}
			}
		}
	}
}

func draw(msets []msetData) {
	var off uint32

	reader, err := os.Open("glenda.jpg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		return
	}
	defer reader.Close()

	m, _, err := image.Decode(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		return
	}
	bounds := m.Bounds()

	for j := 0; j < len(msets); j++ {
		mset := msets[j]
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, _ := m.At(x, y).RGBA()
				off = (mset.fb.stride * uint32(y)) + (uint32(x) * 4)
				val := uint32((uint32(r) << 16) | (uint32(g) << 8) | uint32(b))
				*(*uint32)(unsafe.Pointer(&mset.fb.data[off])) = val
			}
		}
	}

	time.Sleep(10 * time.Second)
}

func clearFramebuffer(fb framebuffer) {
	// Clear the framebuffer by setting all bytes to 0
	for i := uint64(0); i < fb.size; i++ {
		fb.data[i] = 0
	}
}

func destroyFramebuffer(modeset *mode.SimpleModeset, mset msetData, file *os.File) error {
	handle := mset.fb.handle
	data := mset.fb.data
	fb := mset.fb

	err := unix.Munmap(data)
	if err != nil {
		return fmt.Errorf("Failed to munmap memory: %s\n", err.Error())
	}
	err = mode.RmFB(file, fb.id)
	if err != nil {
		return fmt.Errorf("Failed to remove frame buffer: %s\n", err.Error())
	}

	err = mode.DestroyDumb(file, handle)
	if err != nil {
		return fmt.Errorf("Failed to destroy dumb buffer: %s\n", err.Error())
	}
	return modeset.SetCrtc(mset.mode, mset.savedCrtc)
}

func cleanup(modeset *mode.SimpleModeset, msets []msetData, file *os.File) {
	for _, mset := range msets {
		destroyFramebuffer(modeset, mset, file)
	}

}

func main() {
	file, err := drm.OpenCard(0)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		return
	}
	defer file.Close()
	if !drm.HasDumbBuffer(file) {
		fmt.Printf("drm device does not support dumb buffers")
		return
	}
	modeset, err := mode.NewSimpleModeset(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}

	var msets []msetData
	for _, mod := range modeset.Modesets {
		framebuf, err := createFramebuffer(file, &mod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
			cleanup(modeset, msets, file)
			return
		}

		// save current CRTC of this mode to restore at exit
		savedCrtc, err := mode.GetCrtc(file, mod.Crtc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: Cannot get CRTC for connector %d: %s", mod.Conn, err.Error())
			cleanup(modeset, msets, file)
			return
		}
		// change the mode
		err = mode.SetCrtc(file, mod.Crtc, framebuf.id, 0, 0, &mod.Conn, 1, &mod.Mode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot set CRTC for connector %d: %s", mod.Conn, err.Error())
			cleanup(modeset, msets, file)
			return
		}
		msets = append(msets, msetData{
			mode:      &mod,
			fb:        framebuf,
			savedCrtc: savedCrtc,
		})
	}
	if len(msets) > 0 {
		props, err := mode.GetCrtc(file, msets[0].mode.Crtc)
		if err != nil {
			fmt.Printf("Error getting CRTC properties: %v\n", err)
		} else {
			fmt.Printf("CRTC Properties: %+v\n", props)
		}
	}

	//caps := mode.GetCap(file)
	//fmt.Printf("DRM Device Capabilities: %+v\n", caps)

	//renderLoop(file, msets)
	cleanup(modeset, msets, file)
}
