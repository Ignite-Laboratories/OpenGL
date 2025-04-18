package main

// #cgo CFLAGS: -I/usr/include
// #cgo LDFLAGS: -lEGL -lGLESv2 -lbcm_host
// #include <bcm_host.h>
// #include <EGL/egl.h>
// #include <EGL/eglext.h>
// #include <GLES3/gl31.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"github.com/go-gl/gl/v3.1/gles2"
	"runtime"
	"unsafe"
)

var (
	display C.EGLDisplay
	surface C.EGLSurface
	context C.EGLContext
	width   int
	height  int
)

func init() {
	runtime.LockOSThread()
}

func initEGL() error {
	// Initialize Broadcom host
	C.bcm_host_init()

	// Get display size
	var displayWidth, displayHeight C.uint32_t
	C.graphics_get_display_size(0, &displayWidth, &displayHeight)
	width = int(displayWidth)
	height = int(displayHeight)
	fmt.Printf("Display size: %dx%d\n", width, height)

	// Get EGL display
	display = C.eglGetDisplay(C.EGL_DEFAULT_DISPLAY)
	if display == nil {
		return fmt.Errorf("failed to get EGL display")
	}

	// Initialize EGL
	var major, minor C.EGLint
	if C.eglInitialize(display, &major, &minor) == C.EGL_FALSE {
		return fmt.Errorf("failed to initialize EGL")
	}
	fmt.Printf("EGL version: %d.%d\n", major, minor)

	// Configure EGL
	configAttribs := []C.EGLint{
		C.EGL_RED_SIZE, 8,
		C.EGL_GREEN_SIZE, 8,
		C.EGL_BLUE_SIZE, 8,
		C.EGL_ALPHA_SIZE, 8,
		C.EGL_DEPTH_SIZE, 24,
		C.EGL_SURFACE_TYPE, C.EGL_WINDOW_BIT,
		C.EGL_CONFORMANT, C.EGL_OPENGL_ES3_BIT,
		C.EGL_RENDERABLE_TYPE, C.EGL_OPENGL_ES3_BIT,
		C.EGL_NONE,
	}

	var config C.EGLConfig
	var numConfigs C.EGLint
	if C.eglChooseConfig(display, &configAttribs[0], &config, 1, &numConfigs) == C.EGL_FALSE {
		return fmt.Errorf("failed to choose EGL config")
	}

	// Create the native window
	var dstRect, srcRect C.VC_RECT_T
	dstRect.x = 0
	dstRect.y = 0
	dstRect.width = C.int32_t(width)
	dstRect.height = C.int32_t(height)

	srcRect.x = 0
	srcRect.y = 0
	srcRect.width = C.int32_t(width << 16)
	srcRect.height = C.int32_t(height << 16)

	var dispmanDisplay C.DISPMANX_DISPLAY_HANDLE_T
	var dispmanUpdate C.DISPMANX_UPDATE_HANDLE_T
	var dispmanElement C.DISPMANX_ELEMENT_HANDLE_T

	dispmanDisplay = C.vc_dispmanx_display_open(0)
	dispmanUpdate = C.vc_dispmanx_update_start(0)

	var alpha C.VC_DISPMANX_ALPHA_T
	alpha._type = C.DISPMANX_FLAGS_ALPHA_FIXED_ALL_PIXELS
	alpha.opacity = 255
	alpha.mask = 0

	dispmanElement = C.vc_dispmanx_element_add(
		dispmanUpdate,
		dispmanDisplay,
		0, &dstRect,
		0, &srcRect,
		C.DISPMANX_PROTECTION_NONE,
		&alpha,
		nil, 0)

	C.vc_dispmanx_update_submit_sync(dispmanUpdate)

	// Create window surface
	var nativewindow C.EGL_DISPMANX_WINDOW_T
	nativewindow.element = dispmanElement
	nativewindow.width = C.uint32_t(width)
	nativewindow.height = C.uint32_t(height)

	surface = C.eglCreateWindowSurface(display, config, C.EGLNativeWindowType(unsafe.Pointer(&nativewindow)), nil)
	if surface == nil {
		return fmt.Errorf("failed to create window surface")
	}

	// Create OpenGL ES context
	contextAttribs := []C.EGLint{
		C.EGL_CONTEXT_CLIENT_VERSION, 3,
		C.EGL_CONTEXT_MINOR_VERSION, 1,
		C.EGL_NONE,
	}

	context = C.eglCreateContext(display, config, nil, &contextAttribs[0])
	if context == nil {
		return fmt.Errorf("failed to create EGL context")
	}

	// Make context current
	if C.eglMakeCurrent(display, surface, surface, context) == C.EGL_FALSE {
		return fmt.Errorf("failed to make context current")
	}

	return nil
}

func createShaderProgram() (uint32, error) {
	vertexShader := `#version 310 es
    layout(location = 0) in vec3 position;
    layout(location = 1) in vec3 color;
    out vec3 fragColor;
    
    void main() {
        gl_Position = vec4(position, 1.0);
        fragColor = color;
    }`

	fragmentShader := `#version 310 es
    precision mediump float;
    in vec3 fragColor;
    out vec4 outColor;
    
    void main() {
        outColor = vec4(fragColor, 1.0);
    }`

	// Compile vertex shader
	vs := gles2.CreateShader(gles2.VERTEX_SHADER)
	csource, free := gles2.Strs(vertexShader)
	gles2.ShaderSource(vs, 1, csource, nil)
	free()
	gles2.CompileShader(vs)
	var status int32
	gles2.GetShaderiv(vs, gles2.COMPILE_STATUS, &status)
	if status == gles2.FALSE {
		var logLength int32
		gles2.GetShaderiv(vs, gles2.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength)
		gles2.GetShaderInfoLog(vs, logLength, nil, &log[0])
		return 0, fmt.Errorf("vertex shader compilation failed: %s", string(log))
	}

	// Compile fragment shader
	fs := gles2.CreateShader(gles2.FRAGMENT_SHADER)
	csource, free = gles2.Strs(fragmentShader)
	gles2.ShaderSource(fs, 1, csource, nil)
	free()
	gles2.CompileShader(fs)
	gles2.GetShaderiv(fs, gles2.COMPILE_STATUS, &status)
	if status == gles2.FALSE {
		var logLength int32
		gles2.GetShaderiv(fs, gles2.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength)
		gles2.GetShaderInfoLog(fs, logLength, nil, &log[0])
		return 0, fmt.Errorf("fragment shader compilation failed: %s", string(log))
	}

	// Create and link program
	program := gles2.CreateProgram()
	gles2.AttachShader(program, vs)
	gles2.AttachShader(program, fs)
	gles2.LinkProgram(program)
	gles2.GetProgramiv(program, gles2.LINK_STATUS, &status)
	if status == gles2.FALSE {
		var logLength int32
		gles2.GetProgramiv(program, gles2.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength)
		gles2.GetProgramInfoLog(program, logLength, nil, &log[0])
		return 0, fmt.Errorf("program link failed: %s", string(log))
	}

	gles2.DeleteShader(vs)
	gles2.DeleteShader(fs)

	return program, nil
}

func main() {
	if err := initEGL(); err != nil {
		fmt.Printf("Failed to initialize EGL: %v\n", err)
		return
	}
	defer C.eglTerminate(display)

	if err := gles2.Init(); err != nil {
		fmt.Printf("Failed to initialize GLES: %v\n", err)
		return
	}

	// Print OpenGL ES version info
	fmt.Printf("OpenGL ES Version: %s\n", gles2.GoStr(gles2.GetString(gles2.VERSION)))
	fmt.Printf("GLSL ES Version: %s\n", gles2.GoStr(gles2.GetString(gles2.SHADING_LANGUAGE_VERSION)))

	program, err := createShaderProgram()
	if err != nil {
		fmt.Printf("Failed to create shader program: %v\n", err)
		return
	}
	defer gles2.DeleteProgram(program)

	// Create vertex data for a colorful triangle
	vertices := []float32{
		// Position (X, Y, Z)    Color (R, G, B)
		0.0, 0.5, 0.0, 1.0, 0.0, 0.0, // Top vertex (red)
		-0.5, -0.5, 0.0, 0.0, 1.0, 0.0, // Bottom left vertex (green)
		0.5, -0.5, 0.0, 0.0, 0.0, 1.0, // Bottom right vertex (blue)
	}

	// Create and bind VAO
	var vao uint32
	gles2.GenVertexArrays(1, &vao)
	gles2.BindVertexArray(vao)
	defer gles2.DeleteVertexArrays(1, &vao)

	// Create and bind VBO
	var vbo uint32
	gles2.GenBuffers(1, &vbo)
	gles2.BindBuffer(gles2.ARRAY_BUFFER, vbo)
	gles2.BufferData(gles2.ARRAY_BUFFER, len(vertices)*4, gles2.Ptr(vertices), gles2.STATIC_DRAW)
	defer gles2.DeleteBuffers(1, &vbo)

	// Position attribute
	gles2.VertexAttribPointer(0, 3, gles2.FLOAT, false, 6*4, gles2.PtrOffset(0))
	gles2.EnableVertexAttribArray(0)
	// Color attribute
	gles2.VertexAttribPointer(1, 3, gles2.FLOAT, false, 6*4, gles2.PtrOffset(3*4))
	gles2.EnableVertexAttribArray(1)

	// Main render loop
	for {
		// Clear the screen
		gles2.ClearColor(0.2, 0.3, 0.3, 1.0)
		gles2.Clear(gles2.COLOR_BUFFER_BIT)

		// Draw the triangle
		gles2.UseProgram(program)
		gles2.BindVertexArray(vao)
		gles2.DrawArrays(gles2.TRIANGLES, 0, 3)

		// Swap buffers
		C.eglSwapBuffers(display, surface)
	}
}
