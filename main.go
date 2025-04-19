package main

import (
	"fmt"
	"github.com/NeowayLabs/drm"
	"github.com/NeowayLabs/drm/ioctl"
	"github.com/NeowayLabs/drm/mode"
	"golang.org/x/sys/unix"
	_ "image/jpeg"
	"os"
	"os/signal"
	"time"
	"unsafe"
)

const (
	DRM_MODE_PAGE_FLIP_EVENT = 0x01
	DRM_MODE_PAGE_FLIP_ASYNC = 0x02
	DRM_IOCTL_MODE_PAGE_FLIP = 0xc048644d
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

	bufferSet struct {
		front     framebuffer
		back      framebuffer
		mode      *mode.Modeset
		savedCrtc *mode.Crtc
	}

	pageFlipEvent struct {
		crtcID   uint32
		count    uint32
		tv_sec   uint32
		tv_usec  uint32
		reserved uint32
	}
)

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

func createBufferSet(file *os.File, dev *mode.Modeset) (*bufferSet, error) {
	// Create two framebuffers
	front, err := createFramebuffer(file, dev)
	if err != nil {
		return nil, fmt.Errorf("failed to create front buffer: %v", err)
	}

	back, err := createFramebuffer(file, dev)
	if err != nil {
		return nil, fmt.Errorf("failed to create back buffer: %v", err)
	}

	// Save current CRTC
	savedCrtc, err := mode.GetCrtc(file, dev.Crtc)
	if err != nil {
		return nil, fmt.Errorf("cannot get CRTC: %v", err)
	}

	// Clear both buffers initially
	clearFramebuffer(front)
	clearFramebuffer(back)

	// Set initial CRTC with front buffer
	err = mode.SetCrtc(file, dev.Crtc, front.id, 0, 0, &dev.Conn, 1, &dev.Mode)
	if err != nil {
		return nil, fmt.Errorf("cannot set CRTC: %v", err)
	}

	// Wait a bit to ensure the CRTC is properly set
	time.Sleep(50 * time.Millisecond)

	return &bufferSet{
		front:     front,
		back:      back,
		mode:      dev,
		savedCrtc: savedCrtc,
	}, nil
}

func checkDeviceCapabilities(file *os.File) error {
	caps, err := mode.GetCap(file)
	if err != nil {
		return fmt.Errorf("failed to get device capabilities: %v", err)
	}

	if caps.AsyncPageFlip == 0 {
		fmt.Println("Warning: Device doesn't support async page flips")
	}

	return nil
}

func pageFlip(file *os.File, crtcID uint32, fbID uint32) error {
	type pageFlipData struct {
		crtc_id   uint32
		fb_id     uint32
		flags     uint32
		reserved  uint32
		user_data uint64
	}

	req := &pageFlipData{
		crtc_id: crtcID,
		fb_id:   fbID,
		flags:   DRM_MODE_PAGE_FLIP_EVENT,
	}

	err := ioctl.Do(uintptr(file.Fd()),
		uintptr(DRM_IOCTL_MODE_PAGE_FLIP),
		uintptr(unsafe.Pointer(req)))
	if err != nil {
		return fmt.Errorf("page flip ioctl failed: %v", err)
	}
	return nil
}

func renderLoop(file *os.File, buffers []*bufferSet) {
	running := true

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		running = false
	}()

	// Event handling goroutine
	eventChan := make(chan bool, 1)
	go func() {
		buf := make([]byte, unsafe.Sizeof(pageFlipEvent{}))
		for running {
			_, err := unix.Read(int(file.Fd()), buf)
			if err != nil {
				if err != unix.EAGAIN {
					fmt.Fprintf(os.Stderr, "DRM event read failed: %v\n", err)
				}
				continue
			}
			eventChan <- true
		}
	}()

	for running {
		for _, buffer := range buffers {
			// Draw to back buffer
			t := time.Now().UnixNano() / 1000000
			for y := uint32(0); y < uint32(buffer.mode.Height); y++ {
				for x := uint32(0); x < uint32(buffer.mode.Width); x++ {
					off := (buffer.back.stride * y) + (x * 4)
					val := ((x + uint32(t)) ^ y) & 0xFF
					color := (val << 16) | (val << 8) | val
					*(*uint32)(unsafe.Pointer(&buffer.back.data[off])) = color
				}
			}

			// Request page flip
			err := pageFlip(file, buffer.mode.Crtc, buffer.back.id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Page flip request failed: %v\n", err)
				continue
			}

			// Wait for the flip to complete
			<-eventChan

			// Swap buffers
			buffer.front, buffer.back = buffer.back, buffer.front
		}
	}
}

func cleanup(buffers []*bufferSet, file *os.File) {
	for _, buffer := range buffers {
		// Restore original CRTC
		mode.SetCrtc(file, buffer.mode.Crtc, 0, 0, 0, nil, 0, nil)

		// Clean up both framebuffers
		destroyFramebuffer(buffer.front, file)
		destroyFramebuffer(buffer.back, file)
	}
}

func destroyFramebuffer(fb framebuffer, file *os.File) {
	unix.Munmap(fb.data)
	mode.RmFB(file, fb.id)
	mode.DestroyDumb(file, fb.handle)
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

	// Check device capabilities
	if err := checkDeviceCapabilities(file); err != nil {
		fmt.Printf("error checking capabilities: %s", err.Error())
		return
	}

	// Set the file descriptor to non-blocking mode for event reading
	unix.SetNonblock(int(file.Fd()), true)

	modeset, err := mode.NewSimpleModeset(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}

	var buffers []*bufferSet
	for _, mod := range modeset.Modesets {
		bufferSet, err := createBufferSet(file, &mod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
			cleanup(buffers, file)
			return
		}
		buffers = append(buffers, bufferSet)
	}

	//renderLoop(file, buffers)
	cleanup(buffers, file)
}
