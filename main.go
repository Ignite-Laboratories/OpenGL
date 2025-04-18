package main

// #cgo CFLAGS: -I/usr/include -I/usr/include/libdrm
// #cgo LDFLAGS: -lEGL -lGLESv2 -lgbm -ldrm
// #include <EGL/egl.h>
// #include <EGL/eglext.h>
// #include <GLES3/gl31.h>
// #include <gbm.h>
// #include <libdrm/drm.h>
// #include <xf86drm.h>
// #include <xf86drmMode.h>
// #include <fcntl.h>
// #include <unistd.h>
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
	// Open DRM device
	fd := C.open(C.CString("/dev/dri/card0"), C.O_RDWR)
	if fd < 0 {
		return fmt.Errorf("failed to open DRM device")
	}
	defer C.close(fd)

	// Create GBM device
	gbmDevice := C.gbm_create_device(fd)
	if gbmDevice == nil {
		return fmt.Errorf("failed to create GBM device")
	}
	defer C.gbm_device_destroy(gbmDevice)

	// Get the default connector
	resources := C.drmModeGetResources(fd)
	if resources == nil {
		return fmt.Errorf("failed to get DRM resources")
	}
	defer C.drmModeFreeResources(resources)

	var connector *C.drmModeConnector
	for i := 0; i < int(resources.count_connectors); i++ {
		conn := C.drmModeGetConnector(fd, C.uint32_t(*((*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(resources.connectors)) + uintptr(i)*4)))))
		if conn.connection == C.DRM_MODE_CONNECTED {
			connector = conn
			break
		}
		C.drmModeFreeConnector(conn)
	}
	if connector == nil {
		return fmt.Errorf("no connected connector found")
	}
	defer C.drmModeFreeConnector(connector)

	// Get the preferred mode
	mode := (*C.drmModeModeInfo)(unsafe.Pointer(connector.modes))
	width = int(mode.hdisplay)
	height = int(mode.vdisplay)

	// Create GBM surface
	gbmSurface := C.gbm_surface_create(
		gbmDevice,
		C.uint32_t(width),
		C.uint32_t(height),
		C.GBM_FORMAT_XRGB8888,
		C.GBM_BO_USE_SCANOUT|C.GBM_BO_USE_RENDERING)
	if gbmSurface == nil {
		return fmt.Errorf("failed to create GBM surface")
	}
	defer C.gbm_surface_destroy(gbmSurface)

	// Get EGL display
	display = C.eglGetDisplay(C.EGLDisplay(unsafe.Pointer(gbmDevice)))
	if display == nil {
		return fmt.Errorf("failed to get EGL display")
	}

	// Initialize EGL
	var major, minor C.EGLint
	if C.eglInitialize(display, &major, &minor) == C.EGL_FALSE {
		return fmt.Errorf("failed to initialize EGL")
	}

	// Configure EGL
	configAttribs := []C.EGLint{
		C.EGL_SURFACE_TYPE, C.EGL_WINDOW_BIT,
		C.EGL_RED_SIZE, 8,
		C.EGL_GREEN_SIZE, 8,
		C.EGL_BLUE_SIZE, 8,
		C.EGL_ALPHA_SIZE, 0,
		C.EGL_RENDERABLE_TYPE, C.EGL_OPENGL_ES3_BIT,
		C.EGL_NONE,
	}

	var config C.EGLConfig
	var numConfigs C.EGLint
	if C.eglChooseConfig(display, &configAttribs[0], &config, 1, &numConfigs) == C.EGL_FALSE {
		return fmt.Errorf("failed to choose EGL config")
	}

	// Create EGL surface
	surface = C.eglCreateWindowSurface(display, config, C.EGLNativeWindowType(gbmSurface), nil)
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
