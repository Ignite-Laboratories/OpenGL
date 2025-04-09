package main

import (
	"fmt"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

var Alive = true
var id uint64 = 0
var Windows = make(map[uint64]*WindowControl, 0)

func NextID() uint64 {
	return atomic.AddUint64(&id, 1)
}

type WindowControl struct {
	ID       uint64
	WindowID uint32
	Window   *sdl.Window
	Alive    bool
}

func main() {
	runtime.LockOSThread()

	// Initialize SDL
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		log.Fatalf("Failed to initialize SDL: %v", err)
	}
	defer sdl.Quit()

	// Set OpenGL attributes
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3) // OpenGL ES major version 3
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 1) // OpenGL ES minor version 2
	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES)
	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
	sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, 24)

	go RenderLoop(CreateWindow())
	go RenderLoop(CreateWindow())
	go RenderLoop(CreateWindow())
	go RenderLoop(CreateWindow())
	go RenderLoop(CreateWindow())
	go RenderLoop(CreateWindow())
	go RenderLoop(CreateWindow())

	for Alive {
		EventPoll()
	}
	time.Sleep(time.Millisecond * 250)
}

func EventPoll() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch e := event.(type) {
		case *sdl.QuitEvent:
			// Global quit event (e.g., user hits Ctrl+C or closes the app entirely)
			fmt.Println("Received QuitEvent. Shutting down all windows.")
			for _, control := range Windows {
				control.Alive = false
			}
			Alive = false
			return

		case *sdl.WindowEvent:
			// Handle specific window close events
			if e.Event == sdl.WINDOWEVENT_CLOSE {
				HandleWindowEvent(e.WindowID)
			}
		}
	}
}

func HandleWindowEvent(windowID uint32) {
	fmt.Printf("Window %d requested close.\n", windowID)
	for _, control := range Windows {
		if control.WindowID == windowID {
			control.Alive = false
			time.Sleep(time.Millisecond)
			control.Window.Destroy()
		}
	}

	if len(Windows) == 0 {
		Alive = false
	}
}

func CreateWindow() *WindowControl {
	// Create an SDL window
	windowWidth, windowHeight := 800, 600
	window, err := sdl.CreateWindow(
		"SDL2 OpenGL ES 3.2 Example",
		sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		int32(windowWidth), int32(windowHeight),
		sdl.WINDOW_OPENGL|sdl.WINDOW_RESIZABLE,
	)
	if err != nil {
		log.Fatalf("Failed to create SDL window: %v", err)
	}

	w := &WindowControl{}
	w.ID = NextID()
	w.WindowID, _ = window.GetID()
	w.Window = window
	w.Alive = true

	Windows[w.ID] = w
	return w
}

func RenderLoop(ctrl *WindowControl) {
	runtime.LockOSThread()
	defer delete(Windows, ctrl.ID)

	// Create OpenGL context
	glContext, err := ctrl.Window.GLCreateContext()
	if err != nil {
		log.Fatalf("Failed to create OpenGL context: %v", err)
	}
	defer sdl.GLDeleteContext(glContext)

	// Enable VSync
	if err := sdl.GLSetSwapInterval(0); err != nil {
		log.Printf("Failed to set VSync: %v", err)
	}

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		log.Fatalf("Failed to initialize OpenGL: %v", err)
	}

	log.Printf("OpenGL initialized. Version: %s", gl.GoStr(gl.GetString(gl.VERSION)))

	// Get OpenGL version
	glVersion := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL Version:", glVersion)

	// Get and print extensions
	numExtensions := int32(0)
	gl.GetIntegerv(gl.NUM_EXTENSIONS, &numExtensions)

	for i := int32(0); i < numExtensions; i++ {
		extension := gl.GoStr(gl.GetStringi(gl.EXTENSIONS, uint32(i)))
		if strings.Contains(extension, "geometry") {
			fmt.Println("Found geometry-related extension:", extension)
		}
	}

	// Main render loop
	for ctrl.Alive {
		// Clear the screen with a color
		gl.ClearColor(0.2, 0.3, 0.4, 1.0) // RGB color
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Swap buffers (present the rendered frame)
		ctrl.Window.GLSwap()
	}
}
