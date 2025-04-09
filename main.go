package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	// Lock the OS thread for OpenGL context
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
	defer window.Destroy()

	RenderLoop(window)

	for Alive {

	}
}

var Alive = true

func RenderLoop(window *sdl.Window) {
	// Create OpenGL context
	glContext, err := window.GLCreateContext()
	if err != nil {
		log.Fatalf("Failed to create OpenGL context: %v", err)
	}
	defer sdl.GLDeleteContext(glContext)

	// Enable VSync
	if err := sdl.GLSetSwapInterval(1); err != nil {
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
	for Alive {
		// Handle SDL events
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				Alive = false
			}
		}

		// Clear the screen with a color
		gl.ClearColor(0.2, 0.3, 0.4, 1.0) // RGB color
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Swap buffers (present the rendered frame)
		window.GLSwap()

		// Wait a bit (~60 FPS)
		time.Sleep(16 * time.Millisecond)
	}
}
